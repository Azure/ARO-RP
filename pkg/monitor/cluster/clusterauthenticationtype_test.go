package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitClusterAuthenticationType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetrics := mock_metrics.NewMockEmitter(ctrl)

	tests := []struct {
		name                string
		useWorkloadIdentity bool
		expectMetric        map[string]string
	}{
		{
			name:                "Authentication type: Managed Identity",
			useWorkloadIdentity: true,
			expectMetric: map[string]string{
				"type": "managedIdentity",
			},
		},
		{
			name:                "Authentication type: Cluster Service Principal",
			useWorkloadIdentity: false,
			expectMetric: map[string]string{
				"type": "servicePrincipal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			}

			if tt.useWorkloadIdentity {
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = &api.PlatformWorkloadIdentityProfile{}
				doc.OpenShiftCluster.Properties.ServicePrincipalProfile = nil
			} else {
				doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile = nil
				doc.OpenShiftCluster.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{}
			}

			mon := &Monitor{
				doc: doc,
				m:   mockMetrics,
				log: logrus.NewEntry(logrus.New()),
				wg:  &sync.WaitGroup{},
			}

			mockMetrics.EXPECT().EmitGauge(authenticationTypeMetricsTopic, int64(1), tt.expectMetric).Times(1)

			err := mon.emitClusterAuthenticationType(context.Background())
			require.NoError(t, err)
		})
	}
}
