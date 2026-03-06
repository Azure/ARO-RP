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

	sdkazmetrics "github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armnetwork "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armnetwork"
	mock_azmetrics "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azmetrics"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const (
	lbTestSubscriptionID  = "00000000-0000-0000-0000-000000000000"
	lbTestResourceGroup   = "resourceGroupCluster"
	lbTestResourceGroupID = "/subscriptions/" + lbTestSubscriptionID + "/resourcegroups/" + lbTestResourceGroup
	lbTestInfraID         = "infra"
)

func newLBTestDoc(resourceGroupID, infraID string, arch api.ArchitectureVersion) *api.OpenShiftClusterDocument {
	return &api.OpenShiftClusterDocument{
		Key: "testkey",
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				InfraID: infraID,
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: resourceGroupID,
				},
				ArchitectureVersion: arch,
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
		mockMetrics func(*mock_azmetrics.MockMetricsClient)
		wantOutput  []interface{}
		wantErr     string
		wantLogs    []testlog.ExpectedLogEntry
	}{
		{
			name:       "nil clients returns descriptive entry without panic",
			doc:        newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			wantOutput: []interface{}{"load balancer or metrics client missing"},
		},
		{
			name:        "empty ResourceGroupID returns error",
			doc:         newLBTestDoc("", lbTestInfraID, api.ArchitectureVersionV2),
			mockLB:      func(m *mock_armnetwork.MockLoadBalancersClient) {},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {},
			wantErr:     "invalid cluster resource group ID",
		},
		{
			name: "LB Get failure returns error",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{}, errors.New("lb explod"))
			},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {},
			wantErr:     "lb explod",
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("failed to get load balancer"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "nil lb.ID skips metrics query without panic",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{},
					}, nil)
			},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^Load Balancer infra-internal - `),
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
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {
				m.EXPECT().QueryResources(gomock.Any(), lbTestSubscriptionID, "Microsoft.Network/loadBalancers",
					[]string{"DipAvailability", "VipAvailability"},
					gomock.Any(), gomock.Any()).
					Return(sdkazmetrics.QueryResourcesResponse{}, errors.New("metrics explod"))
			},
			wantErr: "metrics explod",
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^Load Balancer infra-internal - `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.ErrorLevel),
					"msg":   gomega.Equal("failed to query health probe metrics for load balancer"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "constant metric value logs one line",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {
				m.EXPECT().QueryResources(gomock.Any(), lbTestSubscriptionID, "Microsoft.Network/loadBalancers",
					[]string{"DipAvailability", "VipAvailability"},
					gomock.Any(), gomock.Any()).
					Return(sdkazmetrics.QueryResourcesResponse{
						MetricResults: sdkazmetrics.MetricResults{
							Values: []sdkazmetrics.MetricData{{
								Values: []sdkazmetrics.Metric{{
									Name:       &sdkazmetrics.LocalizableString{Value: pointerutils.ToPtr("DipAvailability")},
									TimeSeries: []sdkazmetrics.TimeSeriesElement{{
										Data: []sdkazmetrics.MetricValue{
											{TimeStamp: &t0, Average: pointerutils.ToPtr(100.0)},
											{TimeStamp: &t1, Average: pointerutils.ToPtr(100.0)},
											{TimeStamp: &t2, Average: pointerutils.ToPtr(100.0)},
										},
									}},
								}},
							}},
						},
					}, nil)
			},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^Load Balancer infra-internal - `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability  2024-01-15T10:00:00Z: 100%"),
					"lb":    gomega.Equal(lbName),
				},
			},
		},
		{
			name: "metric value change logs two segments",
			doc:  newLBTestDoc(lbTestResourceGroupID, lbTestInfraID, api.ArchitectureVersionV2),
			mockLB: func(m *mock_armnetwork.MockLoadBalancersClient) {
				m.EXPECT().Get(gomock.Any(), lbTestResourceGroup, lbName, nil).
					Return(armnetwork.LoadBalancersClientGetResponse{
						LoadBalancer: armnetwork.LoadBalancer{ID: pointerutils.ToPtr(lbID)},
					}, nil)
			},
			mockMetrics: func(m *mock_azmetrics.MockMetricsClient) {
				m.EXPECT().QueryResources(gomock.Any(), lbTestSubscriptionID, "Microsoft.Network/loadBalancers",
					[]string{"DipAvailability", "VipAvailability"},
					gomock.Any(), gomock.Any()).
					Return(sdkazmetrics.QueryResourcesResponse{
						MetricResults: sdkazmetrics.MetricResults{
							Values: []sdkazmetrics.MetricData{{
								Values: []sdkazmetrics.Metric{{
									Name:       &sdkazmetrics.LocalizableString{Value: pointerutils.ToPtr("DipAvailability")},
									TimeSeries: []sdkazmetrics.TimeSeriesElement{{
										Data: []sdkazmetrics.MetricValue{
											{TimeStamp: &t0, Average: pointerutils.ToPtr(100.0)},
											{TimeStamp: &t1, Average: pointerutils.ToPtr(0.0)},
											{TimeStamp: &t2, Average: pointerutils.ToPtr(0.0)},
										},
									}},
								}},
							}},
						},
					}, nil)
			},
			wantLogs: []testlog.ExpectedLogEntry{
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.MatchRegexp(`^Load Balancer infra-internal - `),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability  2024-01-15T10:00:00Z -> 2024-01-15T10:01:00Z: 100%"),
					"lb":    gomega.Equal(lbName),
				},
				{
					"level": gomega.Equal(logrus.InfoLevel),
					"msg":   gomega.Equal("DipAvailability  2024-01-15T10:01:00Z: 0%"),
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

			m := &manager{
				log: log,
				doc: tt.doc,
			}

			if tt.mockLB != nil && tt.mockMetrics != nil {
				lbClient := mock_armnetwork.NewMockLoadBalancersClient(controller)
				metricsClient := mock_azmetrics.NewMockMetricsClient(controller)
				tt.mockLB(lbClient)
				tt.mockMetrics(metricsClient)
				m.loadBalancers = lbClient
				m.metrics = metricsClient
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
