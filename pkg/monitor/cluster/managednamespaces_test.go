package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/go-test/deep"
	"golang.org/x/exp/slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestGetManagedNamespaces(t *testing.T) {
	ctx := context.Background()

	namespaces := []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cust",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "openshift-monitoring",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "openshift-azure-operator",
			},
		},
		// NOT a managed namespace
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "openshift-gitops",
			},
		},
	}

	_, log := testlog.New()
	ocpclientset := clienthelper.NewWithClient(log, fake.
		NewClientBuilder().
		WithObjects(namespaces...).
		Build())

	mon := &Monitor{
		ocpclientset: ocpclientset,
		queryLimit:   1,
	}

	err := mon.fetchManagedNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}

	slices.Sort(mon.namespacesToMonitor)
	expected := []string{
		"openshift-azure-operator",
		"openshift-monitoring",
	}

	for _, err := range deep.Equal(expected, mon.namespacesToMonitor) {
		t.Error(err)
	}
}
