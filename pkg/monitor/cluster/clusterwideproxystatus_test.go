package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"context"
	"sync"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

// Test cases for emitCWPStatus
func TestEmitCWPStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockMetrics := mock_metrics.NewMockEmitter(ctrl)

	fakeConfigClient := configfake.NewSimpleClientset()

	mon := &Monitor{
		configcli: fakeConfigClient,
		m:         mockMetrics, // Assign the mock emitter here
		log:       logrus.NewEntry(logrus.New()),
		wg:        &sync.WaitGroup{},
	}

	t.Run("no proxy configured", func(t *testing.T) {
		proxy := &configv1.Proxy{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
			Spec:       configv1.ProxySpec{},
		}
		_, _ = fakeConfigClient.ConfigV1().Proxies().Create(context.Background(), proxy, metav1.CreateOptions{})

		mockMetrics.EXPECT().
			EmitGauge("clusterWideProxy.status", int64(1), gomock.Any()).
			Times(1)

		err := mon.emitCWPStatus(context.Background())

		require.NoError(t, err)
	})

	t.Run("missing mandatory no_proxy entries", func(t *testing.T) {
		proxy := &configv1.Proxy{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
			Spec: configv1.ProxySpec{
				NoProxy: "localhost,.svc,.cluster.local",
			},
		}
		_, _ = fakeConfigClient.ConfigV1().Proxies().Create(context.Background(), proxy, metav1.CreateOptions{})

		mockMetrics.EXPECT().
			EmitGauge("clusterWideProxy.status", int64(1), gomock.Any()).
			Times(1)

		err := mon.emitCWPStatus(context.Background())

		require.NoError(t, err)
	})

	t.Run("error fetching proxy configuration", func(t *testing.T) {
		brokenFakeConfigClient := configfake.NewSimpleClientset()
		mon.configcli = brokenFakeConfigClient

		err := mon.emitCWPStatus(context.Background())

		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}
