package node

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"google.golang.org/grpc/peer"
	"p9t.io/kuberboat/pkg/api"
	"p9t.io/kuberboat/pkg/api/core"
	"p9t.io/kuberboat/pkg/apiserver/etcd"
	"p9t.io/kuberboat/pkg/kubelet"
)

// NodeController manages node registry.
type Controller interface {
	// RegisterNode adds the node into the cluster and notifies the node of its successful registry.
	RegisterNode(ctx context.Context, node *core.Node) error
}

type basicController struct {
	nodeManager NodeManager
}

func NewNodeController(nodeManager NodeManager) Controller {
	return &basicController{
		nodeManager: nodeManager,
	}
}

func (bc *basicController) RegisterNode(ctx context.Context, node *core.Node) error {
	// Get node address.
	var workerIP string
	p, _ := peer.FromContext(ctx)
	workerAddr := p.Addr.String()
	if strings.Count(workerAddr, ":") < 2 {
		// IPv4 address
		workerIP = strings.Split(p.Addr.String(), ":")[0]
	} else {
		// IPv6 address
		workerIP = workerAddr[0:strings.LastIndex(workerAddr, ":")]
	}

	node.CreationTimestamp = time.Now()
	node.UUID = uuid.New()
	node.Status.Phase = core.NodePending
	node.Status.Port = kubelet.Port
	node.Status.Address = workerIP
	node.Status.Condition = core.NodeUnavailable

	if err := bc.nodeManager.RegisterNode(node); err != nil {
		glog.Error(err.Error())
		return err
	}

	client := bc.nodeManager.ClientByName(node.Name)
	r, err := client.NotifyRegistered(&core.ApiserverStatus{
		IP:   os.Getenv(api.ApiServerIP),
		Port: core.APISERVER_PORT,
	})
	// If failed to notify worker, rollback registration.
	if err != nil || r.Status != 0 {
		glog.Errorf("cannot notify worker")
		bc.nodeManager.UnregisterNode(node.Name)
		return err
	}

	node.Status.Phase = core.NodeRunning
	node.Status.Condition = core.NodeReady
	err = etcd.Put(fmt.Sprintf("/Nodes/%s", node.Name), node)
	return err
}
