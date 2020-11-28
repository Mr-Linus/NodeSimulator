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
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

const (
	NodeSimFinalizer = "sim.k8s.io/NodeFinal"
)

// NodeSimulatorReconciler reconciles a NodeSimulator object
type NodeSimulatorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sim.k8s.io,resources=nodesimulators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sim.k8s.io,resources=nodesimulators/status,verbs=get;update;patch

func (r *NodeSimulatorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		ctx     = context.Background()
		nodeSim = &simv1.NodeSimulator{}
		nodelist = &v1.NodeList{}
		err     = r.Client.Get(ctx, req.NamespacedName, nodeSim)
	)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Warningf("NodeSim: %v Not Found. ", req.NamespacedName.String())

			return ctrl.Result{}, nil
		} else {
			klog.Errorf("NodeSim: %v Error: %v ", req.NamespacedName.String(), err)
		}
	}

	if nodeSim.GetFinalizers() == nil {
		finalizers := []string{NodeSimFinalizer}
		nodeSim.SetFinalizers(finalizers)
		err := r.Update(ctx,nodeSim)
		if err != nil {
			klog.Errorf("NodeSim %v, Set Finalizers Error: %v",req.NamespacedName.String(),err)
		}
		err = r.Client.Get(ctx, req.NamespacedName, nodeSim)
	}

	if nodeSim.GetDeletionTimestamp() != nil {
		//TODO: Delete

		return ctrl.Result{},nil
	}

	requirement,err := labels.NewRequirement(UniqueLabelKey,selection.Equals,[]string{nodeSim.GetNamespace()+"-"+nodeSim.GetName()})
	if err != nil {
		klog.Errorf("NodeSim: %v Create Requirement Error: %v",req.String(),err)
	}

	nodeSelector := labels.NewSelector()
	nodeSelector.Add(*requirement)
	err = r.Client.List(ctx,nodelist, &client.ListOptions{LabelSelector: nodeSelector})
	if err != nil {
		return ctrl.Result{},err
	}

	// Delete Nodes
	if nodelist.Items != nil && len(nodelist.Items) > nodeSim.Spec.Number {
		for i:= nodeSim.Spec.Number;i < len(nodelist.Items);i++ {
			nodeName := nodeSim.GetNamespace()+"-"+nodeSim.GetName()+strconv.Itoa(i)
			node := &v1.Node{}
			node.SetName(nodeName)
			if err := r.Client.Delete(ctx,node);err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("NodeSim: %v Delete Node: %v Error: %v",req.String(),node,err)
			}
		}
	}

	r.SyncFakeNode(ctx,nodeSim)

	return ctrl.Result{}, nil
}



func (r *NodeSimulatorReconciler) SyncFakeNode(ctx context.Context,nodeSim *simv1.NodeSimulator) {
	// Filter
	if nodeSim.Spec.Number <= 0 {
		return
	}

	nodeTemplate,err := GenNode(nodeSim)
	if  err != nil{
		return
	}

	nodeList := make([]*v1.Node,0)
	// Gen NodeList
	for i := 0; i < nodeSim.Spec.Number; i++ {
		vnode := nodeTemplate.DeepCopy()
		vnode.SetName(nodeSim.GetNamespace()+"-"+nodeSim.GetName()+strconv.Itoa(i))
		nodeList = append(nodeList, vnode)
	}

	Do := func(ctx context.Context,node *v1.Node) {
		fakeNode := &v1.Node{}
		err := r.Client.Get(ctx,types.NamespacedName{
			Name: node.GetName(),
			Namespace: node.GetNamespace(),
		},fakeNode)
		if err != nil && apierrors.IsNotFound(err){
			if err := r.Client.Create(ctx,node); err != nil {
				klog.Errorf("NodeSim: %v/%v Create Node: %v Error: %v ",nodeSim.GetNamespace(),nodeSim.GetName(),node.GetName(),err)
			}
		}else {
			ops := []util.Ops{
				{
					Op: "replace",
					Path: "/spec",
					Value: node.Spec,
				},
				{
					Op: "replace",
					Path: "/status",
					Value: node.Status,
				},
			}

			if err := r.Client.Patch(ctx,node,&util.Patch{PatchOps: ops}); err != nil {
				klog.Errorf("NodeSim: %v/%v Patch Node: %v Error: %v ",nodeSim.GetNamespace(),nodeSim.GetName(),node.GetName(),err)
			}
		}
	}

	util.ParallelizeSyncNode(ctx,5,nodeList,Do)
}





func (r *NodeSimulatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simv1.NodeSimulator{}).
		Complete(r)
}
