package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestEmitClusterOperatorVersion(t *testing.T) {
	ctx := context.Background()

	configcli := configfake.NewSimpleClientset(
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
		})

	m := testmonitor.NewFakeEmitter(t)
	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	err := mon.emitClusterOperatorVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	m.VerifyEmittedMetrics(testmonitor.Metric("clusteroperator.versions", int64(1), map[string]string{
		"name":    "console",
		"version": "4.3.0",
	}))
}
