package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type UnstructuredObj struct {
	obj unstructured.Unstructured
}

func (o *UnstructuredObj) GetObjectKind() schema.ObjectKind {
	return o.obj.GetObjectKind()
}

func (o *UnstructuredObj) DeepCopyObject() kruntime.Object {
	if un := o.obj.DeepCopy(); un != nil {
		return &UnstructuredObj{*un}
	}
	return nil
}

func (o *UnstructuredObj) GroupVersionKind() schema.GroupVersionKind {
	return o.obj.GroupVersionKind()
}

func (o *UnstructuredObj) GetNamespace() string {
	return o.obj.GetNamespace()
}
func (o *UnstructuredObj) GetName() string {
	return o.obj.GetName()
}

func DecodeUnstructured(data []byte) (*UnstructuredObj, error) {
	json, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}
	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(json)
	if err != nil {
		return nil, err
	}
	return &UnstructuredObj{*obj}, nil
}

func isKindUnstructured(groupKind string) bool {
	return strings.HasSuffix(groupKind, ".constraints.gatekeeper.sh")
}
