package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/prometheus/common/model"
	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestIsTargetedAlert(t *testing.T) {
	for _, tt := range []struct {
		name   string
		alert  model.Alert
		expect bool
	}{
		{
			name: "alert with both target and secondary_target labels",
			alert: model.Alert{
				Labels: model.LabelSet{
					"target":           "primary",
					"secondary_target": "secondary",
				},
			},
			expect: true,
		},
		{
			name: "alert with both labels but empty string values",
			alert: model.Alert{
				Labels: model.LabelSet{
					"target":           "",
					"secondary_target": "",
				},
			},
			expect: false,
		},
		{
			name: "alert with only target label",
			alert: model.Alert{
				Labels: model.LabelSet{
					"target": "primary",
				},
			},
			expect: false,
		},
		{
			name: "alert with only secondary_target label",
			alert: model.Alert{
				Labels: model.LabelSet{
					"secondary_target": "secondary",
				},
			},
			expect: false,
		},
		{
			name: "alert with neither target nor secondary_target labels",
			alert: model.Alert{
				Labels: model.LabelSet{
					"severity":  "critical",
					"alertname": "SomeAlert",
				},
			},
			expect: false,
		},
		{
			name: "alert with empty labels",
			alert: model.Alert{
				Labels: model.LabelSet{},
			},
			expect: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := isTargetedAlert(tt.alert)
			if result != tt.expect {
				t.Errorf("expected %v, got %v", tt.expect, result)
			}
		})
	}
}

func TestAggregateAndEmitAlerts(t *testing.T) {
	for _, tt := range []struct {
		name           string
		alerts         []model.Alert
		expectedGauges []struct {
			metric string
			count  int64
			dims   map[string]string
		}
	}{
		{
			name: "regular alerts only",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "Alert1",
						"namespace": "openshift-monitoring",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "Alert2",
						"namespace": "openshift-operators",
						"severity":  "critical",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.alerts",
					count:  1,
					dims: map[string]string{
						"alert":    "Alert1",
						"severity": "warning",
					},
				},
				{
					metric: "prometheus.alerts",
					count:  1,
					dims: map[string]string{
						"alert":    "Alert2",
						"severity": "critical",
					},
				},
			},
		},
		{
			name: "targeted alerts only",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname":        "TargetedAlert1",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "primary-target",
						"secondary_target": "secondary-target",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.targeted.alerts",
					count:  1,
					dims: map[string]string{
						"alert":            "TargetedAlert1",
						"severity":         "warning",
						"target":           "primary-target",
						"secondary_target": "secondary-target",
					},
				},
			},
		},
		{
			name: "mixed regular and targeted alerts",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "RegularAlert",
						"namespace": "openshift-monitoring",
						"severity":  "info",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname":        "TargetedAlert",
						"namespace":        "openshift-operators",
						"severity":         "critical",
						"target":           "foo",
						"secondary_target": "bar",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.alerts",
					count:  1,
					dims: map[string]string{
						"alert":    "RegularAlert",
						"severity": "info",
					},
				},
				{
					metric: "prometheus.targeted.alerts",
					count:  1,
					dims: map[string]string{
						"alert":            "TargetedAlert",
						"severity":         "critical",
						"target":           "foo",
						"secondary_target": "bar",
					},
				},
			},
		},
		{
			name: "multiple alerts with same name get aggregated",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "SameAlert",
						"namespace": "openshift-monitoring",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "SameAlert",
						"namespace": "openshift-monitoring",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "SameAlert",
						"namespace": "openshift-operators",
						"severity":  "warning",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.alerts",
					count:  3,
					dims: map[string]string{
						"alert":    "SameAlert",
						"severity": "warning",
					},
				},
			},
		},
		{
			name: "non-openshift namespace alerts are filtered out",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "CustomerAlert",
						"namespace": "customer-namespace",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "OpenShiftAlert",
						"namespace": "openshift-monitoring",
						"severity":  "warning",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.alerts",
					count:  1,
					dims: map[string]string{
						"alert":    "OpenShiftAlert",
						"severity": "warning",
					},
				},
			},
		},
		{
			name: "ignored alerts are filtered out",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "ImagePruningDisabled",
						"namespace": "openshift-monitoring",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "UsingDeprecatedAPIv1",
						"namespace": "openshift-operators",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "APIRemovedInNextRelease",
						"namespace": "openshift-operators",
						"severity":  "warning",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname": "ValidAlert",
						"namespace": "openshift-monitoring",
						"severity":  "critical",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.alerts",
					count:  1,
					dims: map[string]string{
						"alert":    "ValidAlert",
						"severity": "critical",
					},
				},
			},
		},
		{
			name: "multiple targeted alerts with different targets",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname":        "NodeCondition",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "MemoryPressure",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname":        "NodeCondition",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "node2",
						"secondary_target": "DiskPressure",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname":        "NodeCondition",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "DiskPressure",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.targeted.alerts",
					count:  1,
					dims: map[string]string{
						"alert":            "NodeCondition",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "MemoryPressure",
					},
				},
				{
					metric: "prometheus.targeted.alerts",
					count:  1,
					dims: map[string]string{
						"alert":            "NodeCondition",
						"severity":         "warning",
						"target":           "node2",
						"secondary_target": "DiskPressure",
					},
				},
				{
					metric: "prometheus.targeted.alerts",
					count:  1,
					dims: map[string]string{
						"alert":            "NodeCondition",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "DiskPressure",
					},
				},
			},
		},
		{
			name: "multiple targeted alerts with same targets get aggregated",
			alerts: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname":        "NodeCondition",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "MemoryPressure",
					},
				},
				{
					Labels: model.LabelSet{
						"alertname":        "NodeCondition",
						"namespace":        "openshift-monitoring",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "MemoryPressure",
					},
				},
			},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{
				{
					metric: "prometheus.targeted.alerts",
					count:  2,
					dims: map[string]string{
						"alert":            "NodeCondition",
						"severity":         "warning",
						"target":           "node1",
						"secondary_target": "MemoryPressure",
					},
				},
			},
		},
		{
			name:   "empty alert list",
			alerts: []model.Alert{},
			expectedGauges: []struct {
				metric string
				count  int64
				dims   map[string]string
			}{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)

			mon := &Monitor{
				m: m,
			}

			for _, expected := range tt.expectedGauges {
				m.EXPECT().EmitGauge(expected.metric, expected.count, expected.dims).Times(1)
			}

			mon.aggregateAndEmitAlerts(tt.alerts)
		})
	}
}
