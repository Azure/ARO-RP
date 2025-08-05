package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitDaemonsetStatuses(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		namespaceObject("openshift"),
		namespaceObject("customer"),
		&appsv1.DaemonSet{ // metrics expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name1",
				Namespace: "openshift",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				NumberAvailable:        1,
			},
		}, &appsv1.DaemonSet{ // no metric expected
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name2",
				Namespace: "openshift",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				NumberAvailable:        2,
			},
		}, &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
				Name:      "name2",
				Namespace: "customer",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				NumberAvailable:        1,
			},
		},
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
		queryLimit:   1,
	}

	err := mon.fetchManagedNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.EXPECT().EmitGauge("daemonset.statuses", int64(1), map[string]string{
		"desiredNumberScheduled": strconv.Itoa(2),
		"name":                   "name1",
		"namespace":              "openshift",
		"numberAvailable":        strconv.Itoa(1),
	})

	err = mon.emitDaemonsetStatuses(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
