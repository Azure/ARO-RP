package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/prometheus/common/model"
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
