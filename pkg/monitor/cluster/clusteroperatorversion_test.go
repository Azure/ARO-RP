package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func TestEmitClusterOperatorsVersion(t *testing.T) {
	ctx := context.Background()

	configcli := fake.NewSimpleClientset(&configv1.ClusterOperator{
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
					Version: version.OpenShiftVersion,
				},
			},
		},
	})

	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockInterface(controller)

	mon := &Monitor{
		configcli: configcli,
		m:         m,
	}

	m.EXPECT().EmitGauge("clusteroperators.version", int64(1), map[string]string{
		"name":    "console",
		"version": "4.3.0",
	})

	err := mon.emitClusterOperatorsVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
