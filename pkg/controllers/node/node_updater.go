package node

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"strconv"

	"github.com/NJUPT-ISL/NodeSimulator/pkg/util"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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

// Updater reconciles a NodeSimulator object
type Updater struct {
	Client    client.Client
	ClientSet *kubernetes.Clientset
	Queue     workqueue.RateLimitingInterface
	StopChan  chan struct{}
}

func NewNodeUpdater(updaterClient client.Client, clientSet *kubernetes.Clientset, queue workqueue.RateLimitingInterface, stopChan chan struct{}) (*Updater, error) {
	if updaterClient == nil || queue == nil || stopChan == nil {
		return nil, errors.New("New NodeUpdate Error, parameters contains nil ")
	}
	return &Updater{
		Client:    updaterClient,
		ClientSet: clientSet,
		Queue:     queue,
		StopChan:  stopChan,
	}, nil
}

func (n *Updater) processNextItem() bool {

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

func (n *Updater) runWorker() {
	for n.processNextItem() {
	}
}

func (n *Updater) InitUpdater() {
	for {
		time.Sleep(20 * time.Second)
		nodeList := &v1.NodeList{}
		err := n.Client.List(context.TODO(), nodeList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{ManageLabelKey: ManageLabelValue}),
		})
		if err != nil {
			klog.Errorf("List Node Error: %v", err)
			continue
		}
		if nodeList.Items != nil && len(nodeList.Items) > 0 {
			for _, node := range nodeList.Items {
				getLabels := node.GetLabels()
				if getLabels != nil {
					if v, ok := getLabels[ManageLabelKey]; ok && v == ManageLabelValue {
						n.Queue.Add(node.DeepCopy())
					}
				}
			}
		}
	}
}

func (n *Updater) Run(threadiness int, stopCh chan struct{}) {
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

func (n *Updater) SyncNode(ctx context.Context, node *v1.Node) {

	updateTime := metav1.Time{Time: time.Now()}

	// Update Node Conditions
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

	// update allocate
	nodeName := node.GetName()
	podList, err := n.ClientSet.CoreV1().Pods("").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v", ManageLabelKey, ManageLabelValue),
		FieldSelector: fields.Set{"spec.nodeName": nodeName}.AsSelector().String(),
	})
	if err != nil {
		klog.Errorf("Get Pod from node: %v Error: %v", nodeName, err)
	} else {
		resourceList := node.Status.Capacity.DeepCopy()
		podCount := 0
		if len(podList.Items) > 0 {
			podCount = len(podList.Items)
		}
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				for resourceName, value := range container.Resources.Limits {
					// 当前值
					totalValue := resourceList[resourceName]
					// 当前值减去配额
					totalValue.Sub(value)
					// 更新值
					resourceList[resourceName] = totalValue.DeepCopy()
				}
			}
		}

		if resourceList.Pods() != nil {
			podQua, err := resource.ParseQuantity(strconv.Itoa(podCount))
			if err != nil {
				klog.Errorf("Get Pod Quota node: %v Error: %v", nodeName, err)
			}
			totalValue := resourceList[v1.ResourcePods]
			totalValue.Sub(podQua)
			resourceList[v1.ResourcePods] = totalValue.DeepCopy()
		}

		ops = []util.Ops{
			{
				Op:    "replace",
				Path:  "/status/allocatable",
				Value: resourceList,
			},
		}
		if err = n.Client.Status().Patch(ctx, node, &util.Patch{PatchOps: ops}); err != nil {
			klog.Errorf("Sync Node: %v Error: %v", node.GetName(), err)
		}
	}

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
	err = n.Client.Get(ctx, types.NamespacedName{
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
