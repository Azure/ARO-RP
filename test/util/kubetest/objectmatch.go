package kubetest

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsKubeObject(expected *metav1.PartialObjectMetadata) gomock.Matcher {
	return &kubeMatcher{
		expected: expected,
	}
}

type kubeMatcher struct {
	expected *metav1.PartialObjectMetadata
}

func (k *kubeMatcher) Matches(x interface{}) bool {
	g, ok := x.(client.Object)
	if !ok {
		return false
	}

	return g.GetName() == k.expected.Name && g.GetNamespace() == k.expected.Namespace && g.GetObjectKind().GroupVersionKind().String() == k.expected.GroupVersionKind().String()
}

func (k *kubeMatcher) String() string {
	return k.expected.String()
}
