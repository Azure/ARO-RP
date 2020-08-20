package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// rather than list every GK, just list the ones whose creation really has to be
// brought forward
var createOrder = map[string]int{
	// non-namespaced resources
	"CustomResourceDefinition.apiextensions.k8s.io":    1, // before custom resources
	"ClusterRole.rbac.authorization.k8s.io":            2, // before workload resources
	"ClusterRoleBinding.rbac.authorization.k8s.io":     3, // before workload resources
	"SecurityContextConstraints.security.openshift.io": 4, // before workload resources

	"Namespace": 10, // before namespaced resources

	// namespaced resources
	"ConfigMap":      11, // before workload resources
	"Secret":         12, // before workload resources
	"ServiceAccount": 13, // before workload resources
}

const createOrderMax = 99

// CreateOrder is to be used in a sort.Slice() comparison.  It is to help make
// sure that resources are created in an order that causes a reliable startup.
func CreateOrder(i, j *unstructured.Unstructured) bool {
	io, ok := createOrder[i.GroupVersionKind().GroupKind().String()]
	if !ok {
		io = createOrderMax
	}

	jo, ok := createOrder[j.GroupVersionKind().GroupKind().String()]
	if !ok {
		jo = createOrderMax
	}

	return io < jo
}
