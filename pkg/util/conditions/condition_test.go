package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"reflect"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		input     operatorv1.OperatorCondition

		expected arov1alpha1.ClusterStatus
		wantErr  error
	}{
		{
			name: "noop",
			aroclient: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
			}),
			expected: arov1alpha1.ClusterStatus{
				OperatorVersion: version,
				Conditions:      []operatorv1.OperatorCondition{},
			},
		},
		{
			name: "clean on version change",
			aroclient: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster.String(),
							Status: operatorv1.ConditionFalse,
						},
					},
					OperatorVersion: "?",
				},
			}),
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:   arov1alpha1.InternetReachableFromMaster.String(),
						Status: operatorv1.ConditionFalse,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "noop with condition",
			aroclient: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster.String(),
							Status: operatorv1.ConditionFalse,
						},
					},
				},
			}),
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:   arov1alpha1.InternetReachableFromMaster.String(),
						Status: operatorv1.ConditionFalse,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "change with condition",
			aroclient: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster.String(),
							Status: operatorv1.ConditionFalse,
						},
					},
				},
			}),
			input: operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster.String(),
				Status: operatorv1.ConditionTrue,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster.String(),
						Status:             operatorv1.ConditionTrue,
						LastTransitionTime: transitionTime,
					},
				},
				OperatorVersion: version,
			},
		},
		{
			name: "preserve with condition",
			aroclient: arofake.NewSimpleClientset(&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: objectName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   arov1alpha1.InternetReachableFromMaster.String(),
							Status: operatorv1.ConditionFalse,
						},
						{
							Type:   arov1alpha1.MachineValid.String(),
							Status: operatorv1.ConditionFalse,
						},
					},
				},
			}),
			input: operatorv1.OperatorCondition{
				Type:   arov1alpha1.InternetReachableFromMaster.String(),
				Status: operatorv1.ConditionTrue,
			},
			expected: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					{
						Type:               arov1alpha1.InternetReachableFromMaster.String(),
						Status:             operatorv1.ConditionTrue,
						LastTransitionTime: transitionTime,
					},
					{
						Type:   arov1alpha1.MachineValid.String(),
						Status: operatorv1.ConditionFalse,
					},
				},
				OperatorVersion: version,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			err := SetCondition(ctx, tt.aroclient, &tt.input, role)
			if err != nil && tt.wantErr != nil {
				t.Fatal(err.Error())
			}

			result, err := tt.aroclient.AroV1alpha1().Clusters().Get(ctx, objectName, metav1.GetOptions{})
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
		t          arov1alpha1.CondititionType
		f          func([]operatorv1.OperatorCondition, arov1alpha1.CondititionType) bool
		expect     bool
	}{
		{
			name: "IsTrue - non-existing",
			conditions: []operatorv1.OperatorCondition{
				{
					Type:   arov1alpha1.InternetReachableFromWorker.String(),
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
					Type:   arov1alpha1.InternetReachableFromMaster.String(),
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
					Type:   arov1alpha1.InternetReachableFromMaster.String(),
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
					Type:   arov1alpha1.InternetReachableFromMaster.String(),
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
					Type:   arov1alpha1.InternetReachableFromMaster.String(),
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
