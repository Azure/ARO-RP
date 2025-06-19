package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/yaml"
)

// this is specifically for the Guardrails
type UnstructuredObj struct {
	*unstructured.Unstructured
}

func (o *UnstructuredObj) DeepCopyObject() kruntime.Object {
	if un := o.DeepCopy(); un != nil {
		return &UnstructuredObj{un}
	}
	return nil
}

func DecodeUnstructured(data []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	json, err := yaml.YAMLToJSON(data)
	if err != nil {
		return obj, err
	}
	err = obj.UnmarshalJSON(json)
	return obj, err
}

func isKindUnstructured(groupKind string) bool {
	return strings.HasSuffix(groupKind, ".constraints.gatekeeper.sh")
}
