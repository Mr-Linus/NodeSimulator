package node

import (
	"context"
	"errors"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	cov1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"time"
)

// NodeSimulatorReconciler reconciles a NodeSimulator object
type NodeUpdater struct {
	Client   client.Client
	Queue    workqueue.RateLimitingInterface
	StopChan chan struct{}
}

func NewNodeUpdater(updaterClient client.Client, queue workqueue.RateLimitingInterface, stopChan chan struct{}) (*NodeUpdater, error) {
	if updaterClient == nil || queue == nil || stopChan == nil {
		return nil, errors.New("New NodeUpdate Error, parameters contains nil ")
	}
	return &NodeUpdater{
		Client:   updaterClient,
		Queue:    queue,
		StopChan: stopChan,
	}, nil
}

func (n *NodeUpdater) processNextItem() bool {

	ctx := context.TODO()
	// Wait until there is a new item in the working queue
	key, quit := n.Queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer n.Queue.Done(key)

	if node, ok := key.(*v1.Node); ok {
		n.SyncNode(ctx, node)
	} else {
		klog.Errorf("Key in Queue is not Node Type. ")
	}
	// Invoke the method containing the business logic
	return true
}

func (n *NodeUpdater) runWorker() {
	for n.processNextItem() {
	}
}

func (n *NodeUpdater) InitUpdater() {
	for {
		time.Sleep(30 * time.Second)
		nodeList := &v1.NodeList{}
		err := n.Client.List(context.TODO(), nodeList)
		if err != nil {
			klog.Errorf("List Node Error: %v", err)
			continue
		}
		if nodeList.Items != nil && len(nodeList.Items) > 0 {
			for _, node := range nodeList.Items {
				labels := node.GetLabels()
				if labels != nil {
					if v, ok := labels[ManageLabelKey]; ok && v == ManageLabelValue {
						n.Queue.Add(node.DeepCopy())
					}
				}
			}
		}
	}
}

func (n *NodeUpdater) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer n.Queue.ShutDown()
	klog.Info("Starting Pod controller")

	go n.InitUpdater()

	for i := 0; i < threadiness; i++ {
		go wait.Until(n.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Info("Stopping Node-Updater")
}

func (n *NodeUpdater) SyncNode(ctx context.Context, node *v1.Node) {

	updateTime := metav1.Time{Time: time.Now()}

	// Update Node
	conditions := []v1.NodeCondition{
		{
			LastHeartbeatTime:  updateTime,
			LastTransitionTime: updateTime,
			Message:            KubeletMessage,
			Status:             v1.ConditionTrue,
			Reason:             KubeletReason,
			Type:               v1.NodeReady,
		},
		{
			LastTransitionTime: updateTime,
			LastHeartbeatTime:  updateTime,
			Message:            DiskMessage,
			Status:             v1.ConditionFalse,
			Reason:             DiskReason,
			Type:               OutOfDiskPressure,
		},
		{
			LastHeartbeatTime:  updateTime,
			LastTransitionTime: updateTime,
			Message:            MemoryMessage,
			Status:             v1.ConditionFalse,
			Reason:             MemoryReason,
			Type:               v1.NodeMemoryPressure,
		},
		{
			LastTransitionTime: updateTime,
			LastHeartbeatTime:  updateTime,
			Message:            DiskPressureMessage,
			Status:             v1.ConditionFalse,
			Reason:             DiskPressureReason,
			Type:               v1.NodeDiskPressure,
		},
		{
			LastHeartbeatTime:  updateTime,
			LastTransitionTime: updateTime,
			Message:            RouteMessage,
			Status:             v1.ConditionFalse,
			Reason:             RouteReason,
			Type:               v1.NodeNetworkUnavailable,
		},
	}
	ops := []util.Ops{
		{
			Op:    "replace",
			Path:  "/status/conditions",
			Value: conditions,
		},
	}
	if err := n.Client.Status().Patch(ctx, node, &util.Patch{PatchOps: ops}); err != nil {
		klog.Errorf("Sync Node: %v Error: %v", node.GetName(), err)
	}

	nodeName := node.GetName()
	leasePeriod := int32(40)
	renewTime := metav1.MicroTime{Time: time.Now()}
	lease := &cov1.Lease{}
	newLease := &cov1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.GetName(),
			Namespace: "kube-node-lease",
		},
		Spec: cov1.LeaseSpec{
			HolderIdentity:       &nodeName,
			LeaseDurationSeconds: &leasePeriod,
			RenewTime:            &renewTime,
		},
	}
	err := n.Client.Get(ctx, types.NamespacedName{
		Name:      node.GetName(),
		Namespace: "kube-node-lease",
	}, lease)
	if err != nil && apierrors.IsNotFound(err) {
		err := n.Client.Create(ctx, newLease)
		if err != nil {
			klog.Errorf("Sync Node Lease: %v Error: %v", node.GetName(), err)
		}
		return
	}

	leaseOps := []util.Ops{
		{
			Op:    "replace",
			Path:  "/spec",
			Value: newLease.Spec,
		},
	}
	if err := n.Client.Patch(ctx, lease, &util.Patch{PatchOps: leaseOps}); err != nil {
		klog.Errorf("Sync Node Lease: %v Error: %v", node.GetName(), err)
	}

}
