package dynamichelper

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"

	securityv1 "github.com/openshift/api/security/v1"
)

func TestCreateOrder(t *testing.T) {
	test := []kruntime.Object{
		&corev1.ServiceAccount{},
		&appsv1.Deployment{},
		&corev1.Namespace{},
		&securityv1.SecurityContextConstraints{},
		&corev1.ConfigMap{},
	}

	expect := []kruntime.Object{
		&securityv1.SecurityContextConstraints{},
		&corev1.Namespace{},
		&corev1.ConfigMap{},
		&corev1.ServiceAccount{},
		&appsv1.Deployment{},
	}

	sort.Slice(test, func(i, j int) bool {
		return createOrder(test[i], test[j])
	})

	if !reflect.DeepEqual(expect, test) {
		t.Error(test)
	}
}
