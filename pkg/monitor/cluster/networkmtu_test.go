package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestEmitNetworkMTU(t *testing.T) {
	ctx := context.Background()

	const (
		networkType       string = "OVNKubernetes"
		clusterNetworkMTU int    = 1340
	)

	tests := []struct {
		name           string
		oc             *api.OpenShiftCluster
		networkConfig  *configv1.Network
		expectedMetric metricExpectation
	}{
		{
			name: "MTU cluster test",
			oc: &api.OpenShiftCluster{
				Properties: api.OpenShiftClusterProperties{
					NetworkProfile: api.NetworkProfile{
						MTUSize: api.MTU1500,
					},
				},
			},
			networkConfig: &configv1.Network{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: configv1.NetworkSpec{
					NetworkType: networkType,
				},
				Status: configv1.NetworkStatus{
					ClusterNetworkMTU: clusterNetworkMTU,
				},
			},
			expectedMetric: metricExpectation{
				name:  "network.mtu",
				value: 1,
				labels: map[string]string{
					"mtu":          strconv.Itoa(clusterNetworkMTU),
					"network_type": networkType,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)
			_, log := testlog.New()

			configcli := configfake.NewSimpleClientset(tt.networkConfig)

			mon := &Monitor{
				oc:        tt.oc,
				configcli: configcli,
				m:         m,
				log:       log,
			}

			m.EXPECT().EmitGauge(tt.expectedMetric.name, tt.expectedMetric.value, tt.expectedMetric.labels)

			err := mon.emitNetworkMTU(ctx)
			if err != nil {
				t.Errorf("emitNetworkMTU() error = %v", err)
			}
		})
	}
}

func TestEmitNetworkMTUError(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	m := mock_metrics.NewMockEmitter(controller)
	_, log := testlog.New()

	oc := &api.OpenShiftCluster{
		Properties: api.OpenShiftClusterProperties{
			NetworkProfile: api.NetworkProfile{
				MTUSize: api.MTU1500,
			},
		},
	}

	configcli := configfake.NewSimpleClientset()

	mon := &Monitor{
		oc:        oc,
		configcli: configcli,
		m:         m,
		log:       log,
	}

	err := mon.emitNetworkMTU(ctx)
	if err == nil {
		t.Error("expected error when network config is not found, got nil")
	}

	// Assert that it's a "not found" error
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// metricExpectation represents an expected metric emission
type metricExpectation struct {
	name   string
	value  int64
	labels map[string]string
}
