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
	simv1 "github.com/NJUPT-ISL/NodeSimulator/api/v1"
	"github.com/NJUPT-ISL/NodeSimulator/util"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	err = r.Client.List(ctx,nodelist)
	if err != nil {
		return ctrl.Result{},err
	}

	isCreated := false
	if nodelist.Items != nil && len(nodelist.Items) != 0 {
		for _,node := range nodelist.Items {
			if node.Name == nodeSim.GetNamespace()+"-"+nodeSim.GetName()+"0"{
				isCreated = true
				break
			}
		}
	}

	if !isCreated{
		r.CreatFakeNode(ctx,nodeSim)
	}else {
		r.UpdateFakeNode(ctx,nodeSim)
	}

	return ctrl.Result{}, nil
}



func (r *NodeSimulatorReconciler) CreatFakeNode(ctx context.Context,nodeSim *simv1.NodeSimulator) {
	// Filter
	if nodeSim.Spec.Number <= 0 {
		return
	}

	nodeTemplate,err := GenNode(nodeSim)
	if  err != nil{
		return
	}

	for i := 0; i < nodeSim.Spec.Number; i++ {
		vnode := nodeTemplate.DeepCopy()
		vnode.SetName(nodeSim.GetNamespace()+"-"+nodeSim.GetName()+strconv.Itoa(i))
		if err := r.Client.Create(ctx,vnode); err != nil {
			klog.Errorf("NodeSim: %v/%v Create Node: %v Error: %v ",nodeSim.GetNamespace(),nodeSim.GetName(),vnode.GetName(),err)
		}
	}
}

func (r *NodeSimulatorReconciler) UpdateFakeNode(ctx context.Context,nodeSim *simv1.NodeSimulator) {

	if nodeSim.Spec.Number <= 0 {
		return
	}

	nodeTemplate,err := GenNode(nodeSim)
	if  err != nil{
		return
	}

	for i := 0; i < nodeSim.Spec.Number; i++ {
		vnode := nodeTemplate.DeepCopy()
		vnode.SetName(nodeSim.GetNamespace()+"-"+nodeSim.GetName()+strconv.Itoa(i))

		ops := []util.Ops{
			{
				Op: "replace",
				Path: "/spec",
				Value: vnode.Spec,
			},
			{
				Op: "replace",
				Path: "/status",
				Value: vnode.Status,
			},
		}

		if err := r.Client.Patch(ctx,vnode,&util.Patch{PatchOps: ops}); err != nil {
			klog.Errorf("NodeSim: %v/%v Patch Node: %v Error: %v ",nodeSim.GetNamespace(),nodeSim.GetName(),vnode.GetName(),err)
		}
	}
}

func (r *NodeSimulatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simv1.NodeSimulator{}).
		Complete(r)
}
