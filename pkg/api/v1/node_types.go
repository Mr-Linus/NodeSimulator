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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeSimulatorSpec defines the desired state of NodeSimulator
type NodeSimulatorSpec struct {
	Number    int              `json:"number"`
	PodCIDRs  []string         `json:"podCIDRs,omitempty" protobuf:"bytes,7,opt,name=podCIDRs" patchStrategy:"merge"`
	Taints    []v1.Taint       `json:"taints,omitempty" protobuf:"bytes,5,opt,name=taints"`
	Addresses []v1.NodeAddress `json:"addresses,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,5,rep,name=addresses"`
	Capacity  v1.ResourceList  `json:"capacity,omitempty" protobuf:"bytes,1,rep,name=capacity,casttype=ResourceList,castkey=ResourceName"`
}

// NodeSimulatorStatus defines the observed state of NodeSimulator
type NodeSimulatorStatus struct {
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true

// NodeSimulator is the Schema for the nodesimulators API
type NodeSimulator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeSimulatorSpec   `json:"spec,omitempty"`
	Status NodeSimulatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeSimulatorList contains a list of NodeSimulator
type NodeSimulatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeSimulator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeSimulator{}, &NodeSimulatorList{})
}
