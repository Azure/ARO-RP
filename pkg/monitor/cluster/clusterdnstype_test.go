package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitDNSType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		operatorFlags  api.OperatorFlags
		expectMetric   map[string]string
	}{
		{
			name:          "DNS type: clusterhosted",
			operatorFlags: api.OperatorFlags{"aro.dns.type": "clusterhosted"},
			expectMetric: map[string]string{
				"type": "clusterhosted",
			},
		},
		{
			name:          "DNS type: explicit dnsmasq",
			operatorFlags: api.OperatorFlags{"aro.dns.type": "dnsmasq"},
			expectMetric: map[string]string{
				"type": "dnsmasq",
			},
		},
		{
			name:          "DNS type: empty string defaults to dnsmasq",
			operatorFlags: api.OperatorFlags{"aro.dns.type": ""},
			expectMetric: map[string]string{
				"type": "dnsmasq",
			},
		},
		{
			name:          "DNS type: flag not set defaults to dnsmasq",
			operatorFlags: api.OperatorFlags{},
			expectMetric: map[string]string{
				"type": "dnsmasq",
			},
		},
		{
			name:          "DNS type: nil operator flags defaults to dnsmasq",
			operatorFlags: nil,
			expectMetric: map[string]string{
				"type": "dnsmasq",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetrics := mock_metrics.NewMockEmitter(ctrl)

			oc := &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					OperatorFlags: tt.operatorFlags,
				},
			}

			mon := &Monitor{
				oc:  oc,
				m:   mockMetrics,
				log: logrus.NewEntry(logrus.New()),
			}

			mockMetrics.EXPECT().EmitGauge(dnsTypeMetricsTopic, int64(1), tt.expectMetric).Times(1)

			err := mon.emitDNSType(context.Background())
			require.NoError(t, err)
		})
	}
}
