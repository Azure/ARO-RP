package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	securityv1 "github.com/openshift/api/security/v1"
)

// rather than list every object, just list the ones whose creation really has
// to be brought forward
var createOrderMap = map[reflect.Type]int{
	// non-namespaced resources
	reflect.TypeOf(&extensionsv1.CustomResourceDefinition{}):      -9, // before custom resources
	reflect.TypeOf(&extensionsv1beta1.CustomResourceDefinition{}): -8, // before custom resources
	reflect.TypeOf(&rbacv1.ClusterRole{}):                         -7, // before workload resources
	reflect.TypeOf(&rbacv1.ClusterRoleBinding{}):                  -6, // before workload resources
	reflect.TypeOf(&securityv1.SecurityContextConstraints{}):      -5, // before workload resources

	reflect.TypeOf(&corev1.Namespace{}): -4, // before namespaced resources

	// namespaced resources
	reflect.TypeOf(&corev1.ConfigMap{}):      -3, // before workload resources
	reflect.TypeOf(&corev1.Secret{}):         -2, // before workload resources
	reflect.TypeOf(&corev1.ServiceAccount{}): -1, // before workload resources

	// everything else defaults to 0
}

// createOrder is to be used in a sort.Slice() comparison.  It is to help make
// sure that resources are created in an order that causes a reliable startup.
func createOrder(i, j kruntime.Object) bool {
	return createOrderMap[reflect.TypeOf(i)] < createOrderMap[reflect.TypeOf(j)]
}
