package node

import (
	simv1 "github.com/NJUPT-ISL/NodeSimulator/pkg/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"strconv"
)

func GenNode(nodesim *simv1.NodeSimulator) (*v1.Node, error) {
	labels := make(map[string]string, 0)
	labels[ManageLabelKey] = ManageLabelValue
	labels[UniqueLabelKey] = nodesim.GetNamespace() + "-" + nodesim.GetName()
	podcidr := make([]string, 0)
	podcidr = append(podcidr, nodesim.Spec.PodCidr)
	cpu, err := resource.ParseQuantity(nodesim.Spec.Cpu)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v CPU ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
		return nil, err
	}

	memory, err := resource.ParseQuantity(nodesim.Spec.Memory)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v Memory ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
		return nil, err
	}
	pods, err := resource.ParseQuantity(nodesim.Spec.PodNumber)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v Pods ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
		return nil, err
	}

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: v1.NodeSpec{
			PodCIDR:  nodesim.Spec.PodCidr,
			PodCIDRs: podcidr,
		},
		Status: v1.NodeStatus{
			Capacity: map[v1.ResourceName]resource.Quantity{
				"cpu":    cpu,
				"memory": memory,
				"pods":   pods,
			},
			Allocatable: map[v1.ResourceName]resource.Quantity{
				"cpu":    cpu,
				"memory": memory,
				"pods":   pods,
			},

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

	if nodesim.Spec.Gpu.Number > 0 {
		number, err := resource.ParseQuantity(strconv.Itoa(nodesim.Spec.Gpu.Number))
		if err != nil {
			klog.Errorf("NodeSim: %v/%v GPU Number ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
			return nil, err
		}
		node.Status.Allocatable["gpu/number"] = number
		node.Status.Capacity["gpu/number"] = number

		bandwidth, err := resource.ParseQuantity(nodesim.Spec.Gpu.Bandwidth)
		if err != nil {
			klog.Errorf("NodeSim: %v/%v GPU Bandwidth ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
			return nil, err
		}
		node.Status.Allocatable["gpu/bandwidth"] = bandwidth
		node.Status.Capacity["gpu/bandwidth"] = bandwidth

		memory, err := resource.ParseQuantity(nodesim.Spec.Gpu.Memory)
		if err != nil {
			klog.Errorf("NodeSim: %v/%v GPU Memory ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
			return nil, err
		}
		node.Status.Allocatable["gpu/memory"] = memory
		node.Status.Capacity["gpu/memory"] = memory

		core, err := resource.ParseQuantity(nodesim.Spec.Gpu.Core)
		if err != nil {
			klog.Errorf("NodeSim: %v/%v GPU Core ParseQuantity Error: %v", nodesim.GetNamespace(), nodesim.GetName(), err)
			return nil, err
		}
		node.Status.Allocatable["gpu/core"] = core
		node.Status.Capacity["gpu/core"] = core

	}

	return node, nil
}
