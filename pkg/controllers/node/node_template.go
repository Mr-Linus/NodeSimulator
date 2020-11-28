package node

import (
	simv1 "github.com/NJUPT-ISL/NodeSimulator/pkg/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	ManageLabelKey = "sim.k8s.io/managed"
	ManageLabelValue = "true"
	UniqueLabelKey = "sim.k8s.io/id"
)

func GenNode(nodesim *simv1.NodeSimulator) (*v1.Node,error){
	labels := make(map[string]string,0)
	labels[ManageLabelKey] = ManageLabelValue
	labels[UniqueLabelKey] = nodesim.GetNamespace()+"-"+nodesim.GetName()
	podcidr := make([]string,0)
	podcidr = append(podcidr,nodesim.Spec.PodCidr)

	cpu,err := resource.ParseQuantity(nodesim.Spec.Cpu)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v CPU ParseQuantity Error: %v",nodesim.GetNamespace(),nodesim.GetName(),err)
		return nil, err
	}

	memory,err := resource.ParseQuantity(nodesim.Spec.Memory)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v Memory ParseQuantity Error: %v",nodesim.GetNamespace(),nodesim.GetName(),err)
		return nil, err
	}

	pods,err := resource.ParseQuantity(nodesim.Spec.PodNumber)
	if err != nil {
		klog.Errorf("NodeSim: %v/%v Pods ParseQuantity Error: %v",nodesim.GetNamespace(),nodesim.GetName(),err)
		return nil, err
	}

	node := &v1.Node{
		ObjectMeta:metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: v1.NodeSpec{
			PodCIDR: nodesim.Spec.PodCidr,
			PodCIDRs: podcidr,
		},
		Status: v1.NodeStatus{
			Capacity: map[v1.ResourceName]resource.Quantity{
				"cpu": cpu,
				"memory": memory,
				"pods": pods,
			},
			Allocatable: map[v1.ResourceName]resource.Quantity{
				"cpu": cpu,
				"memory": memory,
				"pods": pods,
			},
		},
	}
	return node, nil
}

