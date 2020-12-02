package pod

import (
	"context"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/controllers/node"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/util"
	scv1 "github.com/NJUPT-ISL/SCV/api/v1"
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

type PodSimReconciler struct {
	Client    client.Client
	ClientSet *kubernetes.Clientset
	Scheme    *runtime.Scheme
}

func (r *PodSimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}

func (r *PodSimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		ctx = context.Background()
		pod = &v1.Pod{}
		err = r.Client.Get(ctx, req.NamespacedName, pod)
	)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Warningf("PodSim: %v Not Found. ", req.NamespacedName.String())
		} else {
			klog.Errorf("PodSim: %v Error: %v ", req.NamespacedName.String(), err)
		}
		return ctrl.Result{}, nil
	}

	labels := pod.GetLabels()
	if labels == nil {
		return ctrl.Result{}, nil
	}
	if v, ok := labels[node.ManageLabelKey]; ok && v == node.ManageLabelValue {
		nodeName := pod.Spec.NodeName
		if nodeName == "" {
			return ctrl.Result{}, nil
		}

		if pod.GetDeletionTimestamp() != nil {
			gracePeriodSeconds := int64(0)
			err = r.ClientSet.CoreV1().Pods(pod.GetNamespace()).Delete(pod.GetName(), &metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds})
			if err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("Delete Pod: %v Error: %v", req.String(), err)
			}
			return ctrl.Result{}, nil
		}

		r.SyncFakePod(pod.DeepCopy())
		r.SyncGPUPod(ctx, nodeName)
	}

	return ctrl.Result{}, nil
}

func (r *PodSimReconciler) SyncFakePod(pod *v1.Pod) {
	updateTime := metav1.Time{Time: time.Now()}
	containerStatusList := make([]v1.ContainerStatus, 0)
	for _, container := range pod.Spec.Containers {
		runningState := &v1.ContainerStateRunning{
			StartedAt: updateTime,
		}
		started := true
		containerStatus := v1.ContainerStatus{
			Name: container.Name,
			State: v1.ContainerState{
				Running: runningState,
			},
			Ready:        true,
			Image:        container.Image,
			Started:      &started,
			RestartCount: 0,
			ImageID:      "docker://sim.k8s.io/podSim/image/" + container.Image,
		}
		containerStatusList = append(containerStatusList, containerStatus)
	}
	conditions := []v1.PodCondition{
		{
			LastProbeTime:      updateTime,
			LastTransitionTime: updateTime,
			Status:             v1.ConditionTrue,
			Type:               v1.PodInitialized,
		},
		{
			LastProbeTime:      updateTime,
			LastTransitionTime: updateTime,
			Status:             v1.ConditionTrue,
			Type:               v1.PodReady,
		},
		{
			LastProbeTime:      updateTime,
			LastTransitionTime: updateTime,
			Status:             v1.ConditionTrue,
			Type:               v1.ContainersReady,
		},
		{
			LastProbeTime:      updateTime,
			LastTransitionTime: updateTime,
			Status:             v1.ConditionTrue,
			Type:               v1.PodScheduled,
		},
	}

	podStatus := v1.PodStatus{
		HostIP:            "10.0.0.1",
		Phase:             v1.PodRunning,
		PodIP:             "10.224.0.1",
		QOSClass:          v1.PodQOSBurstable,
		StartTime:         &updateTime,
		Conditions:        conditions,
		ContainerStatuses: containerStatusList,
	}

	ops := []util.Ops{
		{
			Op:    "replace",
			Path:  "/status",
			Value: podStatus,
		},
	}
	err := r.Client.Status().Patch(context.TODO(), pod, &util.Patch{PatchOps: ops})
	if err != nil {
		klog.Errorf("Pod: %v/%v Patch Status Error: %v", pod.GetNamespace(), pod.GetName(), err)
	}
}

func (r *PodSimReconciler) SyncGPUPod(ctx context.Context, nodeName string) {
	scv := &scv1.Scv{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: nodeName}, scv)
	if err != nil {
		klog.Errorf("Node: %v Get Scv Error: %v", nodeName, err)
		return
	}

	podListWithNode := make([]v1.Pod, 0)
	podList := &v1.PodList{}

	err = r.Client.List(ctx, podList)
	if err != nil {
		klog.Errorf("List Pod Error: %v", err)
		return
	}

	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			podListWithNode = append(podListWithNode, pod)
		}
	}

	cardList := scv.Status.CardList

	for i, card := range cardList {
		cardList[i].FreeMemory = card.TotalMemory
	}

	for _, pod := range podListWithNode {
		mem, _ := strconv.Atoi(pod.GetLabels()["scv/memory"])
		maxCard := 0
		maxMemory := uint64(0)
		for i, card := range cardList {
			if maxMemory < card.FreeMemory {
				maxCard = i
				maxMemory = card.FreeMemory
			}
		}

		cardList[maxCard].FreeMemory -= uint64(mem)
	}

	freeSum := uint64(0)
	for _, card := range cardList {
		freeSum += card.FreeMemory
	}
	scv.Status.FreeMemorySum = freeSum
	scv.Status.CardList = cardList

	ops := []util.Ops{
		{
			Op:    "replace",
			Path:  "/status",
			Value: scv.Status,
		},
	}
	err = r.Client.Patch(context.TODO(), scv, &util.Patch{PatchOps: ops})
	if err != nil {
		klog.Errorf("Scv: %v Patch Status Error: %v", scv.GetName(), err)
	}
}
