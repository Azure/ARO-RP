package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
)

func TestSetControllerConditions(t *testing.T) {
	ctx := context.Background()
	objectName := "cluster"
	version := "unknown"

	kubeclock = clock.NewFakeClock(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	var now = metav1.NewTime(time.Now())
	var past = metav1.NewTime(now.Add(-1 * time.Hour))

	for _, tt := range []struct {
		name     string
		cluster  arov1alpha1.Cluster
		input    ControllerConditions
		expected []operatorv1.OperatorCondition
		wantErr  error
	}{

		{
			name: "sets all provided conditions",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:               arov1alpha1.InternetReachableFromMaster,
							Status:             operatorv1.ConditionFalse,
							LastTransitionTime: now,
						},
					},
					OperatorVersion: version,
				},
			},
			input: ControllerConditions{
				Available: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionTrue,
				},
				Progressing: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				Degraded: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			expected: []operatorv1.OperatorCondition{
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
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
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
					OperatorVersion: version,
				},
			},
			input: ControllerConditions{
				Available: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionTrue,
				},
				Progressing: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				Degraded: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			expected: []operatorv1.OperatorCondition{
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
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
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
					OperatorVersion: version,
				},
			},
			input: ControllerConditions{
				Available: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeAvailable,
					Status: operatorv1.ConditionFalse,
				},
				Progressing: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeProgressing,
					Status: operatorv1.ConditionFalse,
				},
				Degraded: &operatorv1.OperatorCondition{
					Type:   "FakeController" + operatorv1.OperatorStatusTypeDegraded,
					Status: operatorv1.ConditionFalse,
				},
			},
			expected: []operatorv1.OperatorCondition{
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
			clientFake := fake.NewClientBuilder().WithObjects(&tt.cluster).Build()

			err := SetControllerConditions(ctx, clientFake, tt.input)
			if err != tt.wantErr {
				t.Fatalf("wanted error %v, got %v", tt.wantErr, err)
			}

			utilconditions.AssertControllerConditions(t, ctx, clientFake, tt.expected)

			result := &arov1alpha1.Cluster{}
			if err = clientFake.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, result); err != nil {
				t.Fatal(err.Error())
			}

			if diff := cmp.Diff(result.Status.Conditions, tt.expected, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
