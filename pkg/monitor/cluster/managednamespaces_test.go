package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"slices"
	"testing"

	"github.com/go-test/deep"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestGetManagedNamespaces(t *testing.T) {
	ctx := context.Background()
	mon := &Monitor{}
	err := mon.fetchManagedNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(mon.namespacesToMonitor)
	expected := make([]string, len(scopedNamespaces))
	copy(expected, scopedNamespaces)
	slices.Sort(expected)
	for _, err := range deep.Equal(expected, mon.namespacesToMonitor) {
		t.Error(err)
	}
}

func TestEmitPodConditionsMissingNamespace(t *testing.T) {
	ctx := context.Background()

	skippedNamespace := scopedNamespaces[0]

	var objects []client.Object
	for _, ns := range scopedNamespaces {
		objects = append(objects, namespaceObject(ns))
		if ns == skippedNamespace {
			continue
		}
		objects = append(objects, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: ns,
			},
			Spec: corev1.PodSpec{
				NodeName: "fake-node",
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		})
	}

	controller := gomock.NewController(t)
	m := mock_metrics.NewMockEmitter(controller)

	_, log := testlog.New()
	ocpclientset := clienthelper.NewWithClient(log, fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build())

	mon := &Monitor{
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   50,
	}

	err := mon.fetchManagedNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, ns := range scopedNamespaces {
		if ns == skippedNamespace {
			continue
		}
		m.EXPECT().EmitGauge("pod.conditions", int64(1), map[string]string{
			"name":      "test-pod",
			"namespace": ns,
			"nodeName":  "fake-node",
			"status":    "False",
			"type":      "Ready",
		})
	}

	err = mon.emitPodConditions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func namespaceObject(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
