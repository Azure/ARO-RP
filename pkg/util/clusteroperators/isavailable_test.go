package clusteroperators

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsOperatorAvailable(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		availableCondition   configv1.ConditionStatus
		progressingCondition configv1.ConditionStatus
		want                 bool
	}{
		{
			name:                 "Available && Progressing; not available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "Available && !Progressing; available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionFalse,
			want:                 true,
		},
		{
			name:                 "!Available && Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionTrue,
		},
		{
			name:                 "!Available && !Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionFalse,
		},
	} {
		operator := &configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "name",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: tt.availableCondition,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: tt.progressingCondition,
					},
				},
			},
		}
		available := IsOperatorAvailable(operator)
		if available != tt.want {
			t.Error(available)
		}
	}
}

func TestOperatorStatusText(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		availableCondition   configv1.ConditionStatus
		progressingCondition configv1.ConditionStatus
		want                 string
	}{
		{
			name:                 "Available && Progressing; not available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionTrue,
			want:                 "server Available=True, Progressing=True",
		},
		{
			name:                 "Available && !Progressing; available",
			availableCondition:   configv1.ConditionTrue,
			progressingCondition: configv1.ConditionFalse,
			want:                 "server Available=True, Progressing=False",
		},
		{
			name:                 "!Available && Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionTrue,
			want:                 "server Available=False, Progressing=True",
		},
		{
			name:                 "!Available && !Progressing; not available",
			availableCondition:   configv1.ConditionFalse,
			progressingCondition: configv1.ConditionFalse,
			want:                 "server Available=False, Progressing=False",
		},
	} {
		operator := &configv1.ClusterOperator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "server",
			},
			Status: configv1.ClusterOperatorStatus{
				Conditions: []configv1.ClusterOperatorStatusCondition{
					{
						Type:   configv1.OperatorAvailable,
						Status: tt.availableCondition,
					},
					{
						Type:   configv1.OperatorProgressing,
						Status: tt.progressingCondition,
					},
				},
			},
		}
		available := OperatorStatusText(operator)
		if available != tt.want {
			t.Error(available)
		}
	}
}
