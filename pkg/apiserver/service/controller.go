package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/client"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/apiserver/node"
)

type Controller interface {
	// CreateService
	// 		1. Assign cluster IP to the service.
	// 		2. Fill some system-generated properties of the service.
	// 		3. Notify all the nodes in the cluster about the service creation.
	//		4. Modify metadata in component manager.
	CreateService(service *core.Service) error
	// DeleteServiceByName
	// 		1. Notify all the nodes in the cluster about the service deletion.
	// 		2. Modify metadata in component manager.
	DeleteServiceByName(name string) error
	// DeleteAllServices deletes all the services by calling DeleteServiceByName.
	DeleteAllServices() error
	// DescribeServices return all the services and their respective pods.
	DescribeServices(all bool, names []string) ([]*core.Service, [][]string, []string)
}

type basicController struct {
	mtx sync.Mutex
	// componentManager stores the components and the dependencies between them.
	componentManager apiserver.ComponentManager
	// nodeManager provides grpc client to pod controller.
	nodeManager node.NodeManager
	// clusterIPAssigner is responsible for assigning cluster IP to newly created service.
	clusterIPAssigner clusterIPAssigner
}

func NewServiceController(
	componentManager apiserver.ComponentManager,
	nodeManager node.NodeManager,
) Controller {
	clusterIPAssigner, err := NewClusterIPAssigner()
	if err != nil {
		glog.Fatal(err)
	}
	controller := &basicController{
		componentManager:  componentManager,
		nodeManager:       nodeManager,
		clusterIPAssigner: *clusterIPAssigner,
	}
	apiserver.SubscribeToEvent(controller, apiserver.PodReady)
	apiserver.SubscribeToEvent(controller, apiserver.PodDeletion)
	return controller
}

func (c *basicController) CreateService(service *core.Service) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.componentManager.ServiceExistsByName(service.Name) {
		return fmt.Errorf("service already exists: %v", service.Name)
	}

	clusterIP, err := c.clusterIPAssigner.NextClusterIP()
	if err != nil {
		return err
	}
	service.Spec.ClusterIP = clusterIP
	service.UUID = uuid.New()
	service.CreationTimestamp = time.Now()

	selectedPods := c.componentManager.ListPodsByLabelsAndPhase(&service.Spec.Selector, core.PodReady)

	clients := c.nodeManager.Clients()
	errors := make(chan error, len(clients))
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, cli := range clients {
		go func(cli *client.ApiserverClient) {
			defer wg.Done()
			_, err := cli.CreateService(service, selectedPods)
			errors <- err
		}(cli)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}

	// Store service metadata
	if err = etcd.Put(fmt.Sprintf("/Services/Meta/%s", service.Name), service); err != nil {
		return err
	}
	// Store map between service to its pods
	if err = etcd.Put(fmt.Sprintf("/Services/Pods/%s", service.Name), core.GetPodNames(selectedPods)); err != nil {
		return err
	}
	c.componentManager.SetService(service, selectedPods)

	return nil
}

func (c *basicController) DeleteServiceByName(name string) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.componentManager.ServiceExistsByName(name) {
		return fmt.Errorf("no such service: %v", name)
	}

	service := c.componentManager.GetServiceByName(name)
	if service == nil {
		return fmt.Errorf("race condition on service: %v", name)
	}

	clients := c.nodeManager.Clients()
	errors := make(chan error, len(clients))
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, cli := range clients {
		go func(cli *client.ApiserverClient) {
			defer wg.Done()
			_, err := cli.DeleteService(name)
			errors <- err
		}(cli)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}

	deleteServiceInEtcd(service.Name)
	c.componentManager.DeleteServiceByName(name)

	return nil
}

func (c *basicController) DeleteAllServices() error {
	services := c.componentManager.ListServices()
	for _, service := range services {
		if err := c.DeleteServiceByName(service.Name); err != nil {
			return err
		}
	}
	return nil
}

func deleteServiceInEtcd(serviceName string) error {
	// TODO(WindowsXp): maybe we should check delete count and for the following case, it should be 2
	if err := etcd.Delete(fmt.Sprintf("/Services/Meta/%s", serviceName)); err != nil {
		return err
	}
	if err := etcd.Delete(fmt.Sprintf("/Services/Pods/%s", serviceName)); err != nil {
		return err
	}
	return nil
}

func (c *basicController) DescribeServices(all bool, names []string) ([]*core.Service, [][]string, []string) {
	getServicePodNames := func(service *core.Service) []string {
		ret := make([]string, 0)
		pods := c.componentManager.ListPodsByServiceName(service.Name)
		for i := pods.Front(); i != nil; i = i.Next() {
			ret = append(ret, i.Value.(*core.Pod).Name)
		}
		return ret
	}
	servicePods := make([][]string, 0)
	if all {
		services := c.componentManager.ListServices()
		for _, service := range services {
			servicePods = append(servicePods, getServicePodNames(service))
		}
		return services, servicePods, make([]string, 0)
	} else {
		foundServices := make([]*core.Service, 0)
		notFoundServices := make([]string, 0)
		for _, name := range names {
			if !c.componentManager.ServiceExistsByName(name) {
				notFoundServices = append(notFoundServices, name)
			} else {
				service := c.componentManager.GetServiceByName(name)
				if service == nil {
					glog.Errorf("service missing even if cm claims otherwise")
					continue
				}
				foundServices = append(foundServices, service)
				servicePods = append(servicePods, getServicePodNames(service))
			}
		}
		return foundServices, servicePods, notFoundServices
	}
}

func (c *basicController) HandleEvent(event apiserver.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	var err error = nil

	switch event.Type() {
	case apiserver.PodReady:
		podName := event.(*apiserver.PodReadyEvent).PodName
		err = c.handlePodReady(podName)
	case apiserver.PodDeletion:
		pod := event.(*apiserver.PodDeletionEvent).Pod
		err = c.handlePodDeletion(pod)
	}

	if err != nil {
		glog.Error(err)
	}
}

func (c *basicController) handlePodReady(podName string) error {
	pod := c.componentManager.GetPodByName(podName)
	serviceNames := c.componentManager.ListServicesByLabels(&pod.Labels)
	// No service need update
	if len(serviceNames) == 0 {
		return nil
	}

	clients := c.nodeManager.Clients()
	errors := make(chan error, len(clients))
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, cli := range clients {
		go func(cli *client.ApiserverClient) {
			defer wg.Done()
			_, err := cli.AddPodToServices(serviceNames, podName, pod.Status.PodIP)
			errors <- err
		}(cli)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}

	for _, serviceName := range serviceNames {
		c.componentManager.AddPodToService(serviceName, pod)
	}

	return nil
}

func (c *basicController) handlePodDeletion(pod *core.Pod) error {
	serviceNames := c.componentManager.ListServicesByLabels(&pod.Labels)
	// No service need update
	if len(serviceNames) == 0 {
		return nil
	}

	clients := c.nodeManager.Clients()
	errors := make(chan error, len(clients))
	var wg sync.WaitGroup
	wg.Add(len(clients))
	for _, cli := range clients {
		go func(cli *client.ApiserverClient) {
			defer wg.Done()
			_, err := cli.DeletePodFromServices(serviceNames, pod.Name)
			errors <- err
		}(cli)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
