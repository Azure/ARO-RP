package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
)

func TestSetConditions(t *testing.T) {
	ctx := context.Background()

	controllerName := "FakeController"

	now := metav1.NewTime(time.Now())
	past := metav1.NewTime(now.Add(-1 * time.Hour))

	for _, tt := range []struct {
		name  string
		start []operatorv1.OperatorCondition
		input []*operatorv1.OperatorCondition
		want  []operatorv1.OperatorCondition
	}{
		{
			name: "sets all provided conditions",
			start: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
			},
			input: []*operatorv1.OperatorCondition{
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionTrue,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			want: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
			},
		},
		{
			name: "if condition exists and status matches, does not update",
			start: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: past,
				},
			},
			input: []*operatorv1.OperatorCondition{
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionTrue,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			want: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: past,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
			},
		},
		{
			name: "if condition exists and status matches, does not update",
			start: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: past,
				},
			},
			input: []*operatorv1.OperatorCondition{
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionTrue,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			want: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: past,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
			},
		},
		{
			name: "if condition exists and status does not match, updates",
			start: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: past,
				},
			},
			input: []*operatorv1.OperatorCondition{
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			want: []operatorv1.OperatorCondition{
				{
					Type:               arov1alpha1.InternetReachableFromMaster,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
				{
					Type:               "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: now,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := ctrlfake.NewClientBuilder().
				WithObjects(
					&arov1alpha1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: arov1alpha1.SingletonClusterName,
						},
						Status: arov1alpha1.ClusterStatus{
							Conditions:      tt.start,
							OperatorVersion: "unknown",
						},
					},
				).Build()

			controller := AROController{
				Log:    logrus.NewEntry(logrus.StandardLogger()),
				Client: client,
				Name:   controllerName,
			}

			controller.SetConditions(ctx, tt.input...)
			utilconditions.AssertControllerConditions(t, ctx, client, tt.want)
		})
	}
}
