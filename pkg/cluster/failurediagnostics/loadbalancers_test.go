package failurediagnostics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armmonitor "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmonitor"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	lbTestSubscriptionID  = "00000000-0000-0000-0000-000000000000"
	lbTestResourceGroup   = "resourceGroupCluster"
	lbTestResourceGroupID = "/subscriptions/" + lbTestSubscriptionID + "/resourcegroups/" + lbTestResourceGroup
	lbTestInfraID         = "infra"
)

const lbTestClusterID = "/subscriptions/" + lbTestSubscriptionID + "/resourcegroups/resourceGroupRP/providers/Microsoft.RedHatOpenShift/openShiftClusters/cluster"

func newLBTestDoc(resourceGroupID, infraID string) *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key: "testkey",
		OpenShiftCluster: &api.OpenShiftCluster{
			ID: lbTestClusterID,
			Properties: api.OpenShiftClusterProperties{
				InfraID: infraID,
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
			},
		},
	}
}

func TestLogLoadBalancers(t *testing.T) {
	lbID := "/subscriptions/" + lbTestSubscriptionID + "/resourcegroups/" + lbTestResourceGroup + "/providers/Microsoft.Network/loadBalancers/infra-internal"
	lbName := lbTestInfraID + "-internal"

	t0 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 1, 15, 10, 1, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 10, 2, 0, 0, time.UTC)

	for _, tt := range []struct {
		name        string
		doc         *api.OpenShiftClusterDocument
		mockLB      func(*mock_armnetwork.MockLoadBalancersClient)
		mockMetrics func(*mock_armmonitor.MockMetricsClient)
		wantOutput  []interface{}
		wantErr     string
		wantLogs    []testlog.ExpectedLogEntry
	}{
		{
			name:       "nil clients returns descriptive entry without panic",
			doc:        newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			wantOutput: []interface{}{"load balancer or metrics client missing"},
		},
		{
			name: "LB Get failure returns error",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{}, errors.New("lb explod"))
			},
			mockMetrics: func(m *mock_armmonitor.MockMetricsClient) {},
			wantErr:     "lb explod",
		},
		{
			name: "nil lb.ID skips metrics query without panic",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{},
					}, nil)
			},
			mockMetrics: func(m *mock_armmonitor.MockMetricsClient) {},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^load balancer infra-internal: `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("load balancer infra-internal has no ID; skipping metrics query"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "metrics query failure returns error",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_armmonitor.MockMetricsClient) {
				m.EXPECT().List(gomock.Any(), lbID, gomock.Any()).
					Return(armmonitor.MetricsClientListResponse{}, errors.New("metrics explod"))
			},
			wantErr: "metrics explod",
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^load balancer infra-internal: `),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "constant metric value logs one line",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_armmonitor.MockMetricsClient) {
				m.EXPECT().List(gomock.Any(), lbID, gomock.Any()).
					Return(armmonitor.MetricsClientListResponse{
						Response: armmonitor.Response{
							Value: []*armmonitor.Metric{{
								Name: &armmonitor.LocalizableString{Value: pointerutils.ToPtr("DipAvailability")},
								Timeseries: []*armmonitor.TimeSeriesElement{{
									Data: []*armmonitor.MetricValue{
										{TimeStamp: &t0, Average: pointerutils.ToPtr(100.0)},
										{TimeStamp: &t1, Average: pointerutils.ToPtr(100.0)},
										{TimeStamp: &t2, Average: pointerutils.ToPtr(100.0)},
									},
								}},
							}},
						},
					}, nil)
			},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^load balancer infra-internal: `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability 2024-01-15T10:00:00Z: 100%"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "metric value change logs two segments",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_armmonitor.MockMetricsClient) {
				m.EXPECT().List(gomock.Any(), lbID, gomock.Any()).
					Return(armmonitor.MetricsClientListResponse{
						Response: armmonitor.Response{
							Value: []*armmonitor.Metric{{
								Name: &armmonitor.LocalizableString{Value: pointerutils.ToPtr("DipAvailability")},
								Timeseries: []*armmonitor.TimeSeriesElement{{
									Data: []*armmonitor.MetricValue{
										{TimeStamp: &t0, Average: pointerutils.ToPtr(100.0)},
										{TimeStamp: &t1, Average: pointerutils.ToPtr(0.0)},
										{TimeStamp: &t2, Average: pointerutils.ToPtr(0.0)},
									},
								}},
							}},
						},
					}, nil)
			},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^load balancer infra-internal: `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability 2024-01-15T10:00:00Z -> 2024-01-15T10:01:00Z: 100%"),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability 2024-01-15T10:01:00Z: 0%"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			hook, log := testlog.New()

			controller := gomock.NewController(t)
			defer controller.Finish()

			mockEnv := mock_env.NewMockInterface(controller)
			mockEnv.EXPECT().Now().AnyTimes().DoAndReturn(time.Now)

			m := &manager{
				log: log,
				doc: tt.doc,
				env: mockEnv,
			}

			if tt.mockLB != nil && tt.mockMetrics != nil {
				lbClient := mock_armnetwork.NewMockLoadBalancersClient(controller)
				metricsClient := mock_armmonitor.NewMockMetricsClient(controller)
				tt.mockLB(lbClient)
				tt.mockMetrics(metricsClient)
				m.loadBalancers = lbClient
				m.armMonitor = metricsClient
			}

			out, err := m.LogLoadBalancers(ctx)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("want error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantOutput != nil {
				for _, d := range deep.Equal(out, tt.wantOutput) {
					t.Error(d)
				}
			}

			if err := testlog.AssertLoggingOutput(hook, tt.wantLogs); err != nil {
				t.Error(err)
			}
		})
	}
}
