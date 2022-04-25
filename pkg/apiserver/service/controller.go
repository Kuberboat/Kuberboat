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
	// 		3. Modify metadata in component manager.
	//		4. Notify all the nodes in the cluster about the service creation.
	CreateService(service *core.Service) error
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
	c.componentManager.SetService(service, selectedPods)

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
	return nil
}
