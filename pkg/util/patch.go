package util

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type Ops struct {
	Op    string      `json:"op,omitempty"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Patch struct {
	PatchOps []Ops
}

func (p Patch) Type() types.PatchType{
	return  types.JSONPatchType
}

func (p *Patch) Data(obj runtime.Object) ([]byte, error){
	return json.Marshal(p.PatchOps)
}