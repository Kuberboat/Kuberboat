package recover

import (
	"container/list"
	"fmt"

	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/apiserver/node"
)

func Recover(nm *node.NodeManager, cm *apiserver.ComponentManager) error {
	// recover all the nodes
	var nodeType core.Node
	rawNodes, err := etcd.Get("/Nodes", nodeType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, rawNode := range rawNodes {
		node := rawNode.(core.Node)
		if err := (*nm).RegisterNode(&node); err != nil {
			return err
		}
	}
	// recover all the pods
	var podType core.Pod
	pods, err := etcd.Get("/Pods", podType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return nil
	}
	nameToPods := make(map[string]*core.Pod)
	for _, rawPod := range pods {
		pod := rawPod.(core.Pod)
		nameToPods[pod.Name] = &pod
		(*cm).SetPod(&pod)
	}
	// recover all the services
	var serviceType core.Service
	rawServices, err := etcd.Get("/Services/Meta", serviceType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, rawService := range rawServices {
		service := rawService.(core.Service)
		var podNames []string
		rawPodNames, err := etcd.Get(fmt.Sprintf("/Services/Pods/%s", service.Name), podNames)
		if err != nil {
			return err
		}
		if len(rawPodNames) != 1 {
			glog.Fatal("service should have only one pod array")
		}
		podNames = rawPodNames[0].([]string)
		servicePods := list.New()
		for _, podName := range podNames {
			pod, ok := nameToPods[podName]
			if !ok {
				glog.Fatal("service has an unknown pod")
			}
			servicePods.PushBack(pod)
		}
		(*cm).SetService(&service, servicePods)
	}
	// recover all the deployments
	var deploymentType core.Deployment
	rawDeployments, err := etcd.Get("/Deployments/Meta", deploymentType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, rawDeployment := range rawDeployments {
		deployment := rawDeployment.(core.Deployment)
		var podNames []string
		rawPodNames, err := etcd.Get(fmt.Sprintf("/Deployments/Pods/%s", deployment.Name), podNames)
		if err != nil {
			return err
		}
		if len(rawPodNames) > 1 {
			glog.Fatal("service should have only one pod array")
		}
		podNames = rawPodNames[0].([]string)
		deploymentPods := list.New()
		for _, podName := range podNames {
			pod, ok := nameToPods[podName]
			if !ok {
				glog.Fatal("deployment has an unknown pod")
			}
			deploymentPods.PushBack(pod)
		}
		(*cm).SetDeployment(&deployment, deploymentPods)
	}
	return nil
}
