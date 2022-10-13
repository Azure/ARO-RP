package cluster

import (
	"context"
	"reflect"
	"testing"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeMetricsEmitter struct {
	Metrics map[string]fakeMetrics
}

type fakeMetrics struct {
	Value int64
	Dims  map[string]string
}

func newfakeMetricsEmitter() *fakeMetricsEmitter {
	m := make(map[string]fakeMetrics)
	return &fakeMetricsEmitter{
		Metrics: m,
	}
}

func (e *fakeMetricsEmitter) EmitGauge(topic string, value int64, dims map[string]string) {
	data := fakeMetrics{
		Value: value,
	}
	if dims != nil {
		data.Dims = dims
	}
	e.Metrics[topic] = data
}

func (e *fakeMetricsEmitter) EmitFloat(topic string, value float64, dims map[string]string) {}

func TestEmitOperatorFlagsAndSupportBanner(t *testing.T) {
	baseCluster := &arov1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: arov1alpha1.SingletonClusterName,
		},
		Spec: arov1alpha1.ClusterSpec{},
	}

	for _, tt := range []struct {
		name                     string
		operatorFlags            arov1alpha1.OperatorFlags
		clusterBanner            arov1alpha1.Banner
		expectFlagsMetricsValue  int64
		expectFlagsMetricsDims   map[string]string
		expectBannerMetricsValue int64
	}{
		{
			name: "cluster without operator flags and activated support banner",
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 0,
		},
		{
			name: "cluster with standard operator flags",
			operatorFlags: arov1alpha1.OperatorFlags{
				"aro.imageconfig.enabled":   "true",
				"aro.dnsmasq.enabled":       "true",
				"aro.genevalogging.enabled": "true",
			},
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 0,
		},
		{
			name: "cluster with non-standard operator flags",
			operatorFlags: arov1alpha1.OperatorFlags{
				"aro.imageconfig.enabled":   "false",
				"aro.dnsmasq.enabled":       "false",
				"aro.genevalogging.enabled": "false",
			},
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue: 1,
			expectFlagsMetricsDims: map[string]string{
				"aro.imageconfig.enabled":   "false",
				"aro.dnsmasq.enabled":       "false",
				"aro.genevalogging.enabled": "false",
			},
			expectBannerMetricsValue: 0,
		},
		{
			name:          "cluster with activated support banner",
			operatorFlags: arov1alpha1.OperatorFlags{},
			clusterBanner: arov1alpha1.Banner{
				Content: arov1alpha1.BannerContactSupport,
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 1,
		},
		{
			name: "cluster with non-standard operator flags and activated support banner",
			operatorFlags: arov1alpha1.OperatorFlags{
				"aro.imageconfig.enabled":   "false",
				"aro.dnsmasq.enabled":       "false",
				"aro.genevalogging.enabled": "false",
			},
			clusterBanner: arov1alpha1.Banner{
				Content: arov1alpha1.BannerContactSupport,
			},
			expectFlagsMetricsValue: 1,
			expectFlagsMetricsDims: map[string]string{
				"aro.imageconfig.enabled":   "false",
				"aro.dnsmasq.enabled":       "false",
				"aro.genevalogging.enabled": "false",
			},
			expectBannerMetricsValue: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.operatorFlags != nil {
				baseCluster.Spec.OperatorFlags = tt.operatorFlags
			}
			baseCluster.Spec.Banner = tt.clusterBanner
			arocli := arofake.NewSimpleClientset(baseCluster)
			fm := newfakeMetricsEmitter()

			mon := &Monitor{
				arocli: arocli,
				m:      fm,
			}

			err := mon.emitOperatorFlagsAndSupportBanner(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if fm.Metrics[operatorFlagMetricsTopic].Value != tt.expectFlagsMetricsValue {
				t.Errorf("incorrect operator flag metrics value, want: %d, got: %d", tt.expectFlagsMetricsValue, fm.Metrics[operatorFlagMetricsTopic].Value)
			}

			if !reflect.DeepEqual(fm.Metrics[operatorFlagMetricsTopic].Dims, tt.expectFlagsMetricsDims) {
				t.Errorf("incorrect operator flag metrics dims, want: %v, got: %v", tt.expectFlagsMetricsDims, fm.Metrics[operatorFlagMetricsTopic].Dims)
			}

			if fm.Metrics[supportBannerMetricsTopic].Value != tt.expectBannerMetricsValue {
				t.Errorf("incorrect support banner metrics value, want: %d, got: %d", tt.expectBannerMetricsValue, fm.Metrics[supportBannerMetricsTopic].Value)
			}
		})
	}
}
