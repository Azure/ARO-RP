package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func generateDefaultFlags() arov1alpha1.OperatorFlags {
	df := make(arov1alpha1.OperatorFlags)
	for k, v := range operator.DefaultOperatorFlags() {
		df[k] = v
	}
	return df
}

func generateNonStandardFlags(nonDefualtFlagNames []string) arov1alpha1.OperatorFlags {
	nsf := make(arov1alpha1.OperatorFlags)
	for k, v := range operator.DefaultOperatorFlags() {
		nsf[k] = v
	}
	for _, n := range nonDefualtFlagNames {
		if nsf[n] == "true" {
			nsf[n] = "false"
		} else {
			nsf[n] = "true"
		}
	}
	return nsf
}

func generateFlagsWithMissingEntries(missingFlagNames []string) arov1alpha1.OperatorFlags {
	mf := make(arov1alpha1.OperatorFlags)
	for k, v := range operator.DefaultOperatorFlags() {
		mf[k] = v
	}
	for _, n := range missingFlagNames {
		delete(mf, n)
	}
	return mf
}

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
		expectBannerMetricsDims  map[string]string
	}{
		{
			name: "cluster without operator flags and activated support banner",
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 0,
			expectBannerMetricsDims:  nil,
		},
		{
			name:          "cluster with standard operator flags",
			operatorFlags: generateDefaultFlags(),
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 0,
			expectBannerMetricsDims:  nil,
		},
		{
			name:          "cluster with non-standard operator flags",
			operatorFlags: generateNonStandardFlags([]string{operator.ImageConfigEnabled, operator.DnsmasqEnabled, operator.GenevaLoggingEnabled, operator.AutosizedNodesEnabled}),
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue: 1,
			expectFlagsMetricsDims: map[string]string{
				operator.ImageConfigEnabled:    operator.FlagFalse,
				operator.DnsmasqEnabled:        operator.FlagFalse,
				operator.GenevaLoggingEnabled:  operator.FlagFalse,
				operator.AutosizedNodesEnabled: operator.FlagFalse,
			},
			expectBannerMetricsValue: 0,
			expectBannerMetricsDims:  nil,
		},
		{
			name:          "cluster with missing operator flags",
			operatorFlags: generateFlagsWithMissingEntries([]string{operator.ImageConfigEnabled, operator.DnsmasqEnabled, operator.GenevaLoggingEnabled, operator.AutosizedNodesEnabled}),
			clusterBanner: arov1alpha1.Banner{
				Content: "",
			},
			expectFlagsMetricsValue: 1,
			expectFlagsMetricsDims: map[string]string{
				operator.ImageConfigEnabled:    "DNE",
				operator.DnsmasqEnabled:        "DNE",
				operator.GenevaLoggingEnabled:  "DNE",
				operator.AutosizedNodesEnabled: "DNE",
			},
			expectBannerMetricsValue: 0,
			expectBannerMetricsDims:  nil,
		},
		{
			name:          "cluster with activated support banner",
			operatorFlags: generateDefaultFlags(),
			clusterBanner: arov1alpha1.Banner{
				Content: arov1alpha1.BannerContactSupport,
			},
			expectFlagsMetricsValue:  0,
			expectFlagsMetricsDims:   nil,
			expectBannerMetricsValue: 1,
			expectBannerMetricsDims:  map[string]string{"msg": "contact support"},
		},
		{
			name:          "cluster with non-standard operator flags and activated support banner",
			operatorFlags: generateNonStandardFlags([]string{operator.ImageConfigEnabled, operator.DnsmasqEnabled, operator.GenevaLoggingEnabled, operator.AutosizedNodesEnabled}),
			clusterBanner: arov1alpha1.Banner{
				Content: arov1alpha1.BannerContactSupport,
			},
			expectFlagsMetricsValue: 1,
			expectFlagsMetricsDims: map[string]string{
				operator.ImageConfigEnabled:    operator.FlagFalse,
				operator.DnsmasqEnabled:        operator.FlagFalse,
				operator.GenevaLoggingEnabled:  operator.FlagFalse,
				operator.AutosizedNodesEnabled: operator.FlagFalse,
			},
			expectBannerMetricsValue: 1,
			expectBannerMetricsDims:  map[string]string{"msg": "contact support"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.operatorFlags != nil {
				baseCluster.Spec.OperatorFlags = tt.operatorFlags
			}
			baseCluster.Spec.Banner = tt.clusterBanner
			arocli := arofake.NewSimpleClientset(baseCluster)
			m := testmonitor.NewFakeEmitter(t)

			mon := &Monitor{
				arocli: arocli,
				m:      m,
			}

			err := mon.emitOperatorFlagsAndSupportBanner(ctx)
			if err != nil {
				t.Fatal(err)
			}

			emittedMetrics := make([]testmonitor.ExpectedMetric, 0)
			if tt.expectFlagsMetricsValue != 0 {
				emittedMetrics = append(emittedMetrics, testmonitor.Metric(operatorFlagMetricsTopic, tt.expectFlagsMetricsValue, tt.expectFlagsMetricsDims))
			}
			if tt.expectBannerMetricsValue != 0 {
				emittedMetrics = append(emittedMetrics, testmonitor.Metric(supportBannerMetricsTopic, tt.expectBannerMetricsValue, tt.expectBannerMetricsDims))
			}
			m.VerifyEmittedMetrics(emittedMetrics...)
		})
	}
}
