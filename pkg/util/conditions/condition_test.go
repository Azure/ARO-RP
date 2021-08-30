package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	arofake "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/fake"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

func TestSetCondition(t *testing.T) {
	ctx := context.Background()
	role := "master"
	objectName := "cluster"
	version := "unknown"

	kubeclock = &clock.FakeClock{}
	var transitionTime metav1.Time = metav1.Time{Time: kubeclock.Now()}

	for _, tt := range []struct {
		name      string
		aroclient aroclient.Interface
		objects   []runtime.Object
		input     *operatorv1.OperatorCondition

		expected arov1alpha1.ClusterStatus
		wantErr  error
	}{
		{
			name: "no condition provided",
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
			}},
			expected: arov1alpha1.ClusterStatus{},
		},
		{
			name: "new condition provided",
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions:      []operatorv1.OperatorCondition{},
					OperatorVersion: version,
				},
			}},
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
			name: "condition provided without status change - only update operator version",
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster,
							Status: operatorv1.ConditionFalse,
						},
					},
					OperatorVersion: "?",
				},
			}},
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
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:               arov1alpha1.InternetReachableFromMaster,
							Status:             operatorv1.ConditionFalse,
							LastTransitionTime: metav1.Time{Time: time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)},
						},
					},
					OperatorVersion: version,
				},
			}},
			input: &operatorv1.OperatorCondition{
				Type:               arov1alpha1.InternetReachableFromMaster,
				Status:             operatorv1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Date(2021, 0, 0, 0, 0, 0, 0, time.UTC)},
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster,
						Status:             operatorv1.ConditionFalse,
						LastTransitionTime: metav1.Time{Time: time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC)},
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "update one of the existing conditions",
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
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
			}},
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
			objects: []runtime.Object{&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   "staleType",
							Status: operatorv1.ConditionTrue,
						},
					},
					OperatorVersion: version,
				},
			}},
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
			client := arofake.NewSimpleClientset(tt.objects...)

			err := SetCondition(ctx, client, tt.input, role)
			if err != nil && tt.wantErr != nil {
				t.Fatal(err.Error())
			}

			result, err := client.AroV1alpha1().Clusters().Get(ctx, objectName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err.Error())
			}

			if !reflect.DeepEqual(result.Status, tt.expected) {
				t.Fatal(cmp.Diff(result.Status, tt.expected))
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
