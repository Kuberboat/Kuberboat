package recover

import (
	"fmt"

	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	kubeletpod "p9t.io/kuberboat/pkg/kubelet/pod"
	kubeproxy "p9t.io/kuberboat/pkg/kubelet/proxy"
	"p9t.io/kuberboat/pkg/kubelet/proxy/types"
)

func Recover(podMetaManager kubeletpod.MetaManager, runtimeManager kubeletpod.RuntimeManager, proxyMetaManager kubeproxy.MetaManager) error {
	// recover kubelet
	var podType core.Pod
	pods, err := etcd.Get("/Kubelet/Pod", podType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, rawPod := range pods {
		pod := rawPod.(core.Pod)
		podMetaManager.AddPod(&pod, true)
		// recover pods' corresponding runtime
		var containerIds []string
		rawContainerIds, err := etcd.Get(fmt.Sprintf("/Kubelet/Runtime/%v/Containers", pod.Name), containerIds)
		if err != nil {
			return err
		}
		if len(rawContainerIds) > 1 {
			glog.Fatal("pod should have only one containerId array")
		}
		containerIds = rawContainerIds[0].([]string)
		runtimeManager.AddPodContainers(&pod, containerIds)
		sandbox, err := etcd.GetRaw(fmt.Sprintf("/Kubelet/Runtime/%v/Sandbox", pod.Name))
		if err != nil {
			return err
		}
		sandboxName := string(sandbox)
		sandboxName = sandboxName[1 : len(sandboxName)-1]
		runtimeManager.AddPodSandBox(&pod, sandboxName, true)
	}
	// recover kubeproxy
	var serviceType types.ServiceNameWithClusterIP
	services, err := etcd.Get("/Kubeproxy/Service/ClusterIP", serviceType, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, rawService := range services {
		service := rawService.(types.ServiceNameWithClusterIP)
		proxyMetaManager.AddServiceClusterIP(service.ServiceName, service.ClusterIP, true)
		var serviceChainType []types.ServiceChain
		rawServiceChains, err := etcd.Get(fmt.Sprintf("/Kubeproxy/Service/ServiceChain/%v", service.ServiceName), serviceChainType)
		if err != nil {
			return err
		}
		serviceChains := rawServiceChains[0].([]types.ServiceChain)
		for _, serviceChain := range serviceChains {
			proxyMetaManager.AddServiceChain(service.ServiceName, &serviceChain, true)
			var podChainType []types.PodChain
			rawPodChains, err := etcd.Get(fmt.Sprintf("/Kubeproxy/ServiceChain/%v", serviceChain.ChainName), podChainType)
			if err != nil {
				return err
			}
			if len(rawPodChains) > 0 {
				podChains := rawPodChains[0].([]types.PodChain)
				for _, podChain := range podChains {
					proxyMetaManager.AddPodChain(serviceChain.ChainName, &podChain, true)
				}
			}
		}
	}
	return nil
}
