package types

import "encoding/json"

type MergedPathMetaProp[T any] struct {
	V    T
	Path string
}

func (p MergedPathMetaProp[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.V)
}

type MergedPathMeta struct {
	Password      MergedPathMetaProp[string] `json:"-"`
	DefaultSort   MergedPathMetaProp[string] `json:"defaultSort"`
	DefaultMode   MergedPathMetaProp[string] `json:"defaultMode"`
	HiddenPattern MergedPathMetaProp[string] `json:"hiddenPattern"`
}

var _ json.Marshaler = MergedPathMetaProp[string]{}
