package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitSummary(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset(&configv1.ClusterVersion{
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
	})

	cli := fake.NewSimpleClientset(&corev1.Node{
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
		})

	mockCreatedAt := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	m := testmonitor.NewFakeEmitter(t)
	mon := &Monitor{
		configcli: configcli,
		cli:       cli,
		m:         m,
		oc: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: api.ProvisioningStateFailed,
				CreatedAt:         mockCreatedAt,
			},
		},
		hourlyRun: true,
	}

	err := mon.emitSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.VerifyEmittedMetrics(
		testmonitor.Metric("cluster.summary", int64(1), map[string]string{
			"actualVersion":     "4.3.0",
			"desiredVersion":    "4.3.3",
			"masterCount":       "1",
			"workerCount":       "2",
			"provisioningState": api.ProvisioningStateFailed.String(),
			"createdAt":         mockCreatedAt.String(),
		}))
}
