package node

import v1 "k8s.io/api/core/v1"

const (
	NodeSimFinalizer = "sim.k8s.io/NodeFinal"

	ManageLabelKey     = "sim.k8s.io/managed"
	ManageLabelValue   = "true"
	UniqueLabelKey     = "sim.k8s.io/id"
	NodeOS             = "linux"
	NodeArch           = "amd64"
	NodeOSImage        = "CentOS Linux 7 (Core)"
	NodeKernel         = "3.10.0.el7.x86_64"
	NodeKubeletVersion = "v1.19.1"
	NodeDockerVersion  = "docker://18.6.3"

	// Condition
	KubeletMessage      = "kubelet is ready."
	DiskMessage         = "kubelet has sufficient disk space available"
	MemoryMessage       = "kubelet has sufficient memory available"
	DiskPressureMessage = "kubelet has no disk pressure"
	RouteMessage        = "RouteController created a route"

	// Reason
	KubeletReason      = "KubeletReady"
	DiskReason         = "KubeletHasSufficientDisk"
	MemoryReason       = "MemoryPressure"
	DiskPressureReason = "KubeletHasNoDiskPressure"
	RouteReason        = "RouteCreated"

	// Type
	OutOfDiskPressure v1.NodeConditionType = "OutOfDisk"
)
