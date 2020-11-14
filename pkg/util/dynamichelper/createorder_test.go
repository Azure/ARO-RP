package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateOrder(t *testing.T) {
	test := []*unstructured.Unstructured{
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ServiceAccount"}},
		{Object: map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment"}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Namespace"}},
		{Object: map[string]interface{}{"apiVersion": "security.openshift.io/v1", "kind": "SecurityContextConstraints"}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap"}},
	}

	expect := []*unstructured.Unstructured{
		{Object: map[string]interface{}{"apiVersion": "security.openshift.io/v1", "kind": "SecurityContextConstraints"}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Namespace"}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap"}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ServiceAccount"}},
		{Object: map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment"}},
	}

	sort.Slice(test, func(i, j int) bool {
		return createOrder(test[i], test[j])
	})

	if !reflect.DeepEqual(expect, test) {
		t.Error(test)
	}
}
