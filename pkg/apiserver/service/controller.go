package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/client"
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
}

type basicController struct {
	// componentManager stores the components and the dependencies between them.
	componentManager apiserver.ComponentManager
	// nodeManager provides grpc client to pod controller.
	nodeManager apiserver.NodeManager
	// clusterIPAssigner is responsible for assigning cluster IP to newly created service.
	clusterIPAssigner clusterIPAssigner
}

func NewServiceController(
	componentManager apiserver.ComponentManager,
	nodeManager apiserver.NodeManager,
) (Controller, error) {
	clusterIPAssigner, err := NewClusterIPAssigner()
	if err != nil {
		return nil, err
	}
	return &basicController{
		componentManager:  componentManager,
		nodeManager:       nodeManager,
		clusterIPAssigner: *clusterIPAssigner,
	}, nil
}

func (c *basicController) CreateService(service *core.Service) error {
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

	c.componentManager.SetService(service, selectedPods)

	return nil
}

func (c *basicController) DeleteServiceByName(name string) error {
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
