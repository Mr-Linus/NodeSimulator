/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"context"
	simv1 "github.com/NJUPT-ISL/NodeSimulator/pkg/api/v1"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/util"
	scv1 "github.com/NJUPT-ISL/SCV/api/v1"
	"github.com/go-logr/logr"
	cov1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

// NodeSimReconciler reconciles a NodeSimulator object
type NodeSimReconciler struct {
	client.Client
	ClientSet *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=sim.k8s.io,resources=nodesimulators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sim.k8s.io,resources=nodesimulators/status,verbs=get;update;patch

func (r *NodeSimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		ctx      = context.Background()
		nodeSim  = &simv1.NodeSimulator{}
		nodeList = &v1.NodeList{}
		err      = r.Client.Get(ctx, req.NamespacedName, nodeSim)
	)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Warningf("NodeSim: %v Not Found. ", req.NamespacedName.String())
		} else {
			klog.Errorf("NodeSim: %v Error: %v ", req.NamespacedName.String(), err)
		}
		return ctrl.Result{}, nil
	}

	// Get Node List
	err = r.Client.List(ctx, nodeList, &client.MatchingLabels{
		ManageLabelKey: ManageLabelValue,
		UniqueLabelKey: nodeSim.GetNamespace() + "-" + nodeSim.GetName(),
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if nodeSim.GetFinalizers() == nil {
		finalizers := []string{NodeSimFinalizer}
		nodeSim.SetFinalizers(finalizers)
		err := r.Update(ctx, nodeSim)
		if err != nil {
			klog.Errorf("NodeSim %v, Set Finalizers Error: %v", req.NamespacedName.String(), err)
		}
		// Get NodeSim
		err = r.Client.Get(ctx, req.NamespacedName, nodeSim)
		if err != nil {
			klog.Errorf("Get NodeSim: %v Error: %v", req.String(), err)
		}
	}

	if nodeSim.GetDeletionTimestamp() != nil {
		if nodeList.Items != nil && len(nodeList.Items) > 0 {
			for _, node := range nodeList.Items {
				// Delete Node
				if err := r.Client.Delete(ctx, node.DeepCopy()); err != nil {
					klog.Errorf("NodeSim: %v Delete Node: %v Error: %v", req.NamespacedName.String(), node.GetName(), err)
				}

				// Delete Node
				scv := &scv1.Scv{}
				err = r.Client.Get(ctx, types.NamespacedName{Name: node.GetName()}, scv)
				if err != nil && !apierrors.IsNotFound(err) {
					klog.Errorf("Get Scv: %v Error: %v", node.GetName(), err)
				}
				if err == nil {
					err = r.Client.Delete(ctx, scv)
					if err != nil {
						klog.Errorf("Delete Scv: %v Error: %v", node.GetName(), err)
					}
				}

				// Delete Node Lease
				nodeLease := &cov1.Lease{}
				nodeLease.SetName(node.GetName())
				nodeLease.SetNamespace("kube-node-lease")
				if err := r.Client.Delete(ctx, nodeLease); err != nil && !apierrors.IsNotFound(err) {
					klog.Errorf("NodeSim: %v Delete Node Lease : %v Error: %v", req.String(), node, err)
				}
			}
		}
		nodeSim.SetFinalizers(nil)
		if err := r.Update(ctx, nodeSim); err != nil {
			klog.Errorf("")
		}

		return ctrl.Result{}, nil
	}

	// Delete Nodes
	if nodeList.Items != nil && len(nodeList.Items) > nodeSim.Spec.Number {
		for i := nodeSim.Spec.Number; i < len(nodeList.Items); i++ {

			//Delete Node
			nodeName := nodeSim.GetNamespace() + "-" + nodeSim.GetName() + "-" + strconv.Itoa(i)
			node := &v1.Node{}
			node.SetName(nodeName)
			if err := r.Client.Delete(ctx, node); err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("NodeSim: %v Delete Node: %v Error: %v", req.String(), node, err)
			}
			// Delete Node Lease
			nodeLease := &cov1.Lease{}
			nodeLease.SetName(nodeName)
			nodeLease.SetNamespace("kube-node-lease")
			if err := r.Client.Delete(ctx, nodeLease); err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("NodeSim: %v Delete Node Lease : %v Error: %v", req.String(), node, err)
			}

			// Delete Scv
			scv := &scv1.Scv{}
			err = r.Client.Get(ctx, types.NamespacedName{Name: node.GetName()}, scv)
			if err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("Get Scv: %v Error: %v", node.GetName(), err)
			}
			if err == nil {
				err = r.Client.Delete(ctx, scv)
				if err != nil {
					klog.Errorf("Delete Scv: %v Error: %v", node.GetName(), err)
				}
			}
		}
	}

	r.SyncFakeNode(ctx, nodeSim)

	return ctrl.Result{}, nil
}

func (r *NodeSimReconciler) SyncFakeNode(ctx context.Context, nodeSim *simv1.NodeSimulator) {
	// Filter
	if nodeSim.Spec.Number <= 0 {
		return
	}

	nodeTemplate, err := GenNode(nodeSim)
	if err != nil {
		return
	}

	nodeList := make([]*v1.Node, 0)
	// Gen NodeList
	for i := 0; i < nodeSim.Spec.Number; i++ {
		vnode := nodeTemplate.DeepCopy()
		vnode.SetName(nodeSim.GetNamespace() + "-" + nodeSim.GetName() + "-" + strconv.Itoa(i))
		nodeList = append(nodeList, vnode)
	}

	SyncNode := func(ctx context.Context, node *v1.Node) {
		fakeNode := &v1.Node{}

		node.Status.Addresses = []v1.NodeAddress{
			{
				Type:    v1.NodeHostName,
				Address: node.GetName(),
			},
		}

		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      node.GetName(),
			Namespace: node.GetNamespace(),
		}, fakeNode)
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, node); err != nil {
				klog.Errorf("NodeSim: %v/%v Create Node: %v Error: %v ", nodeSim.GetNamespace(), nodeSim.GetName(), node.GetName(), err)
			}
		} else {
			specOps := []util.Ops{
				{
					Op:    "replace",
					Path:  "/spec",
					Value: node.Spec,
				},
			}

			if err := r.Client.Patch(ctx, node, &util.Patch{PatchOps: specOps}); err != nil {
				klog.Errorf("NodeSim: %v/%v Patch Node: %v Error: %v ", nodeSim.GetNamespace(), nodeSim.GetName(), node.GetName(), err)
			}

			newNode := fakeNode.DeepCopy()
			newNode.Status.Allocatable = nodeTemplate.Status.Allocatable
			newNode.Status.Capacity = nodeTemplate.Status.Capacity
			newNode.Status.Addresses = node.Status.Addresses
			_, _, err := util.PatchNodeStatus(r.ClientSet.CoreV1(), types.NodeName(node.GetName()), fakeNode, newNode)
			if err != nil {
				klog.Errorf("Patch Node: %v Error: %v", newNode.GetName(), err)
			}
		}
	}

	util.ParallelizeSyncNode(ctx, 5, nodeList, SyncNode)

	SyncNodeGPU := func(ctx context.Context, node *v1.Node) {
		if nodeSim.Spec.Gpu.Number <= 0 {
			return
		}

		curScv := &scv1.Scv{}
		cardList := make([]scv1.Card, 0)

		memSum := uint64(0)

		for i := 0; i < nodeSim.Spec.Gpu.Number; i++ {
			card := scv1.Card{
				ID:          uint(i),
				Health:      "Healthy",
				Model:       "RTX TITAN",
				Power:       250,
				TotalMemory: strToUint64(nodeSim.Spec.Gpu.Memory),
				Clock:       6000,
				FreeMemory:  strToUint64(nodeSim.Spec.Gpu.Memory),
				Core:        strToUint(nodeSim.Spec.Gpu.Core),
				Bandwidth:   strToUint(nodeSim.Spec.Gpu.Bandwidth),
			}
			cardList = append(cardList, card)
			memSum += strToUint64(nodeSim.Spec.Gpu.Memory)
		}

		updateTime := metav1.Time{Time: time.Now()}

		scv := &scv1.Scv{
			ObjectMeta: metav1.ObjectMeta{
				Name: node.GetName(),
			},
			Spec: scv1.ScvSpec{
				UpdateInterval: 1000,
			},
			Status: scv1.ScvStatus{
				CardList:       cardList,
				CardNumber:     uint(nodeSim.Spec.Gpu.Number),
				TotalMemorySum: memSum,
				FreeMemorySum:  memSum,
				UpdateTime:     &updateTime,
			},
		}

		err := r.Client.Get(ctx, types.NamespacedName{
			Name: node.GetName(),
		}, curScv)

		if err != nil {
			if apierrors.IsNotFound(err) {
				if err = r.Client.Create(ctx, scv); err != nil {
					klog.Errorf("Create Scv: %v, Error: %v", node.GetName(), err)
				}
			} else {
				klog.Errorf("Get Scv: %v, Error: %v", node.GetName(), err)
			}

		} else {
			ops := []util.Ops{
				{
					Op:    "replace",
					Path:  "/status",
					Value: scv.Status,
				},
			}

			err = r.Client.Patch(ctx, curScv, &util.Patch{PatchOps: ops})
			if err != nil {
				klog.Errorf("Update Scv: %v, Error: %v", node.GetName(), err)
			}
		}

	}

	util.ParallelizeSyncNode(ctx, 5, nodeList, SyncNodeGPU)
}

func (r *NodeSimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simv1.NodeSimulator{}).
		Complete(r)
}

func strToUint64(str string) uint64 {
	if i, e := strconv.Atoi(str); e != nil {
		return 0
	} else {
		return uint64(i)
	}
}

func strToUint(str string) uint {
	if i, e := strconv.Atoi(str); e != nil {
		return 0
	} else {
		return uint(i)
	}
}
