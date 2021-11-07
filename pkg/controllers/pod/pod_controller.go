package pod

import (
	"context"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/controllers/node"
	"github.com/NJUPT-ISL/NodeSimulator/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type SimReconciler struct {
	Client    client.Client
	ClientSet *kubernetes.Clientset
	Scheme    *runtime.Scheme
}

func (r *SimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}

func (r *SimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
	}

	return ctrl.Result{}, nil
}

func (r *SimReconciler) SyncFakePod(pod *v1.Pod) {
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
