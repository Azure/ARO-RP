package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitSummary(t *testing.T) {
	ctx := context.Background()

	objects := []client.Object{
		&configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Status: configv1.ClusterVersionStatus{
				Desired: configv1.Release{
					Version: "4.3.3",
				},
				History: []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.3.0",
					},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "aro-master-0",
				Labels: map[string]string{
					masterRoleLabel: "",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "aro-node-1",
				Labels: map[string]string{
					workerRoleLabel: "",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "aro-node-2",
				Labels: map[string]string{
					workerRoleLabel: "",
				},
			},
		},
	}

	mockCreatedAt := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

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

		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateFailed,
				CreatedAt:         mockCreatedAt,
			},
		},
		hourlyRun: true,
	}

	m.EXPECT().EmitGauge("cluster.summary", int64(1), map[string]string{
		"actualVersion":      "4.3.0",
		"actualMinorVersion": "4.3",
		"desiredVersion":     "4.3.3",
		"masterCount":        "1",
		"workerCount":        "2",
		"provisioningState":  api.ProvisioningStateFailed.String(),
		"createdAt":          mockCreatedAt.String(),
	})

	err := mon.prefetchClusterVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = mon.emitSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
