package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clockTesting "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestSetCondition(t *testing.T) {
	ctx := context.Background()
	role := "master"
	objectName := "cluster"
	version := "unknown"

	kubeclock = clockTesting.NewFakeClock(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	var transitionTime = metav1.Time{Time: kubeclock.Now()}

	for _, tt := range []struct {
		name    string
		cluster arov1alpha1.Cluster
		input   *operatorv1.OperatorCondition

		expected arov1alpha1.ClusterStatus
		wantErr  string
	}{
		{
			name: "no condition provided",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
			},
			expected: arov1alpha1.ClusterStatus{},
		},
		{
			name: "new condition provided",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions:      []operatorv1.OperatorCondition{},
					OperatorVersion: version,
				},
			},
			input: &operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster,
				Status: operatorv1.ConditionFalse,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster,
						Status:             operatorv1.ConditionFalse,
						LastTransitionTime: transitionTime,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "condition provided without status change - only update operator version",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster,
							Status: operatorv1.ConditionFalse,
						},
					},
					OperatorVersion: "?",
				},
			},
			input: &operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster,
				Status: operatorv1.ConditionFalse,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:   arov1alpha1.InternetReachableFromMaster,
						Status: operatorv1.ConditionFalse,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "condition provided without status change - no update",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:               arov1alpha1.InternetReachableFromMaster,
							Status:             operatorv1.ConditionFalse,
							LastTransitionTime: metav1.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
					OperatorVersion: version,
				},
			},
			input: &operatorv1.OperatorCondition{
				Type:               arov1alpha1.InternetReachableFromMaster,
				Status:             operatorv1.ConditionFalse,
				LastTransitionTime: metav1.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster,
						Status:             operatorv1.ConditionFalse,
						LastTransitionTime: metav1.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "update one of the existing conditions",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster,
							Status: operatorv1.ConditionFalse,
						},
						{
							Type:   arov1alpha1.MachineValid,
							Status: operatorv1.ConditionFalse,
						},
					},
					OperatorVersion: "?",
				},
			},
			input: &operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster,
				Status: operatorv1.ConditionTrue,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster,
						Status:             operatorv1.ConditionTrue,
						LastTransitionTime: transitionTime,
					},
					{
						Type:   arov1alpha1.MachineValid,
						Status: operatorv1.ConditionFalse,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "cleanup stale conditions",
			cluster: arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: objectName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   "staleType",
							Status: operatorv1.ConditionTrue,
						},
					},
					OperatorVersion: version,
				},
			},
			input: &operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster,
				Status: operatorv1.ConditionTrue,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster,
						Status:             operatorv1.ConditionTrue,
						LastTransitionTime: transitionTime,
					},
				},
				OperatorVersion: version,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			clientFake := fake.NewClientBuilder().WithObjects(&tt.cluster).WithStatusSubresource(&tt.cluster).Build()

			err := SetCondition(ctx, clientFake, tt.input, role)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			result := &arov1alpha1.Cluster{}
			if err = clientFake.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, result); err != nil {
				t.Fatal(err.Error())
			}

			// cmp.Diff correctly compares times the same time, but in different timezones
			// unlike reflect.DeepEqual which compares field by field.
			// We need this because fake client marshals and unmarshals objects
			// due to this line[1] in apimachiner time than gets converted to a local time.
			//
			// [1] https://github.com/kubernetes/apimachinery/blob/24bec8a7ae9ed9efe31aa9239cc616d751c2bc69/pkg/apis/meta/v1/time.go#L115
			diff := cmp.Diff(result.Status, tt.expected)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestIsConditions(t *testing.T) {
	for _, tt := range []struct {
		name       string
		conditions []operatorv1.OperatorCondition
		t          string
		f          func([]operatorv1.OperatorCondition, string) bool
		expect     bool
	}{
		{
			name: "IsTrue - non-existing",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromWorker,
					Status: operatorv1.ConditionTrue,
				},
			},
			t:      arov1alpha1.InternetReachableFromMaster,
			f:      IsTrue,
			expect: false,
		},
		{
			name: "IsTrue - true",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromMaster,
					Status: operatorv1.ConditionTrue,
				},
			},
			t:      arov1alpha1.InternetReachableFromMaster,
			f:      IsTrue,
			expect: true,
		},
		{
			name: "IsTrue - false",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromMaster,
					Status: operatorv1.ConditionFalse,
				},
			},
			t:      arov1alpha1.InternetReachableFromMaster,
			f:      IsTrue,
			expect: false,
		},
		{
			name: "IsFalse - true",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromMaster,
					Status: operatorv1.ConditionFalse,
				},
			},
			t:      arov1alpha1.InternetReachableFromMaster,
			f:      IsFalse,
			expect: true,
		},
		{
			name: "IsFalse - false",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromMaster,
					Status: operatorv1.ConditionTrue,
				},
			},
			t:      arov1alpha1.InternetReachableFromMaster,
			f:      IsFalse,
			expect: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.f(tt.conditions, tt.t)
			if result != tt.expect {
				t.Fatalf("expected %t, got %t", tt.expect, result)
			}
		})
	}
}
