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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeSimulatorSpec defines the desired state of NodeSimulator
type NodeSimulatorSpec struct {
	Prefix    string `json:"prefix"`
	Cpu       string `json:"cpu"`
	Memory    string `json:"memory"`
	PodNumber string `json:"podNumber"`
	Number    int    `json:"number"`
	PodCidr   string `json:"podCidr"`
	Gpu       GPU    `json:"gpu,omitempty"`
}

type GPU struct {
	Number    int    `json:"number,omitempty"`
	Memory    string `json:"memory,omitempty"`
	Core      string `json:"core,omitempty"`
	Bandwidth string `json:"bandwidth,omitempty"`
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
