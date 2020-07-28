package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func copyImmutableFields(to, from *unstructured.Unstructured) {
	to.SetResourceVersion(from.GetResourceVersion())
	to.SetCreationTimestamp(from.GetCreationTimestamp())
	to.SetSelfLink(from.GetSelfLink())
	to.SetUID(from.GetUID())
	to.SetGeneration(from.GetGeneration())

	status, found, err := unstructured.NestedMap(from.Object, "status")
	if err == nil && found {
		unstructured.SetNestedMap(to.Object, status, "status")
	}
	if to.GetKind() == "Service" {
		cIP, found, err := unstructured.NestedString(from.Object, "spec", "clusterIP")
		if err == nil && found {
			unstructured.SetNestedField(to.Object, cIP, "spec", "clusterIP")
		}
	}
}
