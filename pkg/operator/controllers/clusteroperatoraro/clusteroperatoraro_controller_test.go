package clusteroperatoraro

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestConditions(t *testing.T) {
	tests := []struct {
		name                 string
		controllerConditions []operatorv1.OperatorCondition
		wantConditions       []configv1.ClusterOperatorStatusCondition
		wantErr              string
	}{
		{
			name:                 "no conditions sets defaults",
			controllerConditions: []operatorv1.OperatorCondition{},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
		{
			name: "All controllers available sets Available=True",
			controllerConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable("ControllerA"),
				utilconditions.ControllerDefaultAvailable("ControllerB"),
				utilconditions.ControllerDefaultAvailable("ControllerC"),
			},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
					Message:            "All is well",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
		{
			name: "Controller not available sets Available=False",
			controllerConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultAvailable("ControllerA"),
				{
					Type:               "ControllerBAvailable",
					Status:             operatorv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "SomeError",
					Message:            "An error occurred",
				},
				utilconditions.ControllerDefaultAvailable("ControllerC"),
			},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "ControllerB_SomeError",
					Message:            "ControllerBAvailable: An error occurred",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
		{
			name: "All controllers not progressing sets Progressing=False",
			controllerConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultProgressing("ControllerA"),
				utilconditions.ControllerDefaultProgressing("ControllerB"),
				utilconditions.ControllerDefaultProgressing("ControllerC"),
			},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
					Message:            "All is well",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
		{
			name: "Controller progressing sets Progressing=True",
			controllerConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultProgressing("ControllerA"),
				{
					Type:               "ControllerBProgressing",
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "SomeProcess",
					Message:            "Something is happening",
				},
				utilconditions.ControllerDefaultProgressing("ControllerC"),
			},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "ControllerB_SomeProcess",
					Message:            "ControllerBProgressing: Something is happening",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
		{
			name: "Controller degraded does NOT set Degraded=True",
			controllerConditions: []operatorv1.OperatorCondition{
				utilconditions.ControllerDefaultDegraded("ControllerA"),
				{
					Type:               "ControllerBDegraded",
					Status:             operatorv1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "SomeProcess",
					Message:            "Something bad is happening",
				},
				utilconditions.ControllerDefaultDegraded("ControllerC"),
			},
			wantConditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "NoData",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "AsExpected",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: tt.controllerConditions,
				},
			}

			co := defaultOperator()
			clientFake := ctrlfake.NewClientBuilder().
				WithObjects(cluster, co).
				WithStatusSubresource(cluster, co).
				Build()

			r := NewReconciler(logrus.NewEntry(logrus.StandardLogger()), clientFake)

			request := ctrl.Request{}
			ctx := context.Background()

			_, err := r.Reconcile(ctx, request)

			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			operator := &configv1.ClusterOperator{}
			if err := clientFake.Get(ctx, types.NamespacedName{Name: clusterOperatorName}, operator); err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt.wantConditions, operator.Status.Conditions, cmpopts.EquateApproxTime(time.Second)); diff != "" {
				t.Error(diff)
			}

			// static checks - these should always be set on the operator resource after every reconcile
			wantVersion := []configv1.OperandVersion{{
				Name:    "operator",
				Version: version.GitCommit,
			}}
			if diff := cmp.Diff(wantVersion, operator.Status.Versions); diff != "" {
				t.Error(diff)
			}

			wantOwnerReference := []metav1.OwnerReference{{
				APIVersion:         arov1alpha1.GroupVersion.Identifier(),
				Kind:               "Cluster",
				Name:               arov1alpha1.SingletonClusterName,
				Controller:         ptr.To(true),
				BlockOwnerDeletion: ptr.To(true),
			}}
			if diff := cmp.Diff(wantOwnerReference, operator.ObjectMeta.OwnerReferences); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func defaultOperator() *configv1.ClusterOperator {
	currentTime := metav1.Now()
	return &configv1.ClusterOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterOperatorName,
		},
		Status: configv1.ClusterOperatorStatus{
			Versions: []configv1.OperandVersion{
				{
					Name:    "operator",
					Version: version.GitCommit,
				},
			},
			Conditions: []configv1.ClusterOperatorStatusCondition{
				{
					Type:               configv1.OperatorAvailable,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: currentTime,
					Reason:             reasonInitializing,
					Message:            "Operator is initializing",
				},
				{
					Type:               configv1.OperatorProgressing,
					Status:             configv1.ConditionTrue,
					LastTransitionTime: currentTime,
					Reason:             reasonInitializing,
					Message:            "Operator is initializing",
				},
				{
					Type:               configv1.OperatorDegraded,
					Status:             configv1.ConditionFalse,
					LastTransitionTime: currentTime,
					Reason:             reasonAsExpected,
				},
			},
		},
	}
}
