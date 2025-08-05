package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitClusterOperatorVersion(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		&configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "console",
			},
			Status: configv1.ClusterOperatorStatus{
				Versions: []configv1.OperandVersion{
					{
						Name:    "operator",
						Version: "4.3.0",
					},
					{
						Name:    "operator-good", // no metrics exected
						Version: "4.3.1",
					},
				},
			},
		},
		&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Status: configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: "4.3.1",
				},
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
		log:          log,
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   1,
	}

	m.EXPECT().EmitGauge("clusteroperator.versions", int64(1), map[string]string{
		"name":    "console",
		"version": "4.3.0",
	})

	err := mon.prefetchClusterVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = mon.emitClusterOperatorVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
