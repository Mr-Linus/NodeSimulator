package node

import (
	simv1 "github.com/NJUPT-ISL/NodeSimulator/pkg/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenNode(nodesim *simv1.NodeSimulator) (*v1.Node, error) {
	labels := nodesim.GetLabels()

	if labels == nil {
		labels = make(map[string]string, 0)
	}

	labels[ManageLabelKey] = ManageLabelValue
	labels[UniqueLabelKey] = nodesim.GetNamespace() + "-" + nodesim.GetName()

	podCidr := ""
	if len(nodesim.Spec.PodCIDRs) > 0 {
		podCidr = nodesim.Spec.PodCIDRs[0]
	}

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: v1.NodeSpec{
			PodCIDR:  podCidr,
			PodCIDRs: nodesim.Spec.PodCIDRs,
			Taints:   nodesim.Spec.Taints,
		},
		Status: v1.NodeStatus{
			Capacity:    nodesim.Spec.Capacity,
			Allocatable: nodesim.Spec.Capacity,
			Addresses:   nodesim.Spec.Addresses,
			NodeInfo: v1.NodeSystemInfo{
				OperatingSystem:         NodeOS,
				Architecture:            NodeArch,
				OSImage:                 NodeOSImage,
				KernelVersion:           NodeKernel,
				KubeletVersion:          NodeKubeletVersion,
				KubeProxyVersion:        NodeKubeletVersion,
				ContainerRuntimeVersion: NodeDockerVersion,
			},
		},
	}
	return node, nil
}
