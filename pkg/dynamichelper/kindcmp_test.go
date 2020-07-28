package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"sort"
	"testing"
)

func TestKindLess(t *testing.T) {
	test := []string{
		"ServiceAccount", "Deployment", "Namespace", "SecurityContextConstraints", "ConfigMap", "Service",
	}
	expect := []string{
		"Namespace", "SecurityContextConstraints", "ServiceAccount", "ConfigMap", "Service", "Deployment",
	}

	sort.Slice(test, func(i, j int) bool {
		return KindLess(test[i], test[j])
	})

	if !reflect.DeepEqual(expect, test) {
		t.Error(test)
	}
}
