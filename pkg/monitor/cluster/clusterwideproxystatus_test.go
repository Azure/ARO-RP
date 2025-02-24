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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestEmitCWPStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetrics := mock_metrics.NewMockEmitter(ctrl)
	fakeConfigClient := configfake.NewSimpleClientset()

	mon := &Monitor{
		configcli: fakeConfigClient,
		m:         mockMetrics,
		log:       logrus.NewEntry(logrus.New()),
		wg:        &sync.WaitGroup{},
	}

	tests := []struct {
		name          string
		proxyConfig   *configv1.Proxy
		expectErr     bool
		expectedError string
		setupMocks    func(*mock_metrics.MockEmitter)
	}{
		{
			name: "no proxy configured",
			proxyConfig: &configv1.Proxy{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec:       configv1.ProxySpec{},
			},
			expectErr:     false,
			expectedError: "",
			setupMocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().
					EmitGauge("clusterWideProxy.status", int64(1), gomock.Any()).
					Times(1)
			},
		},
		{
			name: "missing mandatory no_proxy entries",
			proxyConfig: &configv1.Proxy{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.ProxySpec{
					NoProxy: "localhost,.svc,.cluster.local",
				},
			},
			expectErr:     false,
			expectedError: "",
			setupMocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().
					EmitGauge("clusterWideProxy.status", int64(1), gomock.Any()).
					Times(1)
			},
		},
		{
			name:          "error fetching proxy configuration",
			proxyConfig:   &configv1.Proxy{},
			expectErr:     false,
			expectedError: "",
			setupMocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().
					EmitGauge("clusterWideProxy.status", int64(1), gomock.Any()).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.proxyConfig != nil {
				_, _ = fakeConfigClient.ConfigV1().Proxies().Create(context.Background(), tt.proxyConfig, metav1.CreateOptions{})
			}

			tt.setupMocks(mockMetrics)

			err := mon.emitCWPStatus(context.Background())

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
