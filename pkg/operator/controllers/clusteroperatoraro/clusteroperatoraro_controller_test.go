package clusteroperatoraro

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"github.com/Azure/ARO-RP/pkg/util/version"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
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
			_, log := testlog.LogForTesting(t)

			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Status: arov1alpha1.ClusterStatus{
					Conditions: tt.controllerConditions,
				},
			}
			clientFake := testclienthelper.NewAROFakeClientBuilder(cluster).Build()

			r := NewReconciler(log, clientFake)

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
				Controller:         new(true),
				BlockOwnerDeletion: new(true),
			}}
			if diff := cmp.Diff(wantOwnerReference, operator.OwnerReferences); diff != "" {
				t.Error(diff)
			}
		})
	}
}

// testStatusWriter wraps a StatusWriter to intercept Update calls.
type testStatusWriter struct {
	client.StatusWriter
	updateFunc func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error
}

func (w *testStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if w.updateFunc != nil {
		return w.updateFunc(ctx, obj, opts...)
	}
	return w.StatusWriter.Update(ctx, obj, opts...)
}

// testClient wraps a client.Client, allowing Update and Status().Update interception.
type testClient struct {
	client.Client
	statusWriter client.StatusWriter
	updateFunc   func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

func (c *testClient) Status() client.StatusWriter {
	if c.statusWriter != nil {
		return c.statusWriter
	}
	return c.Client.Status()
}

func (c *testClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if c.updateFunc != nil {
		return c.updateFunc(ctx, obj, opts...)
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestReconcileRetryOnConflict(t *testing.T) {
	t.Run("conflict during metadata Update retries successfully", func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		cluster := &arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
			Status: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					utilconditions.ControllerDefaultAvailable("TestController"),
				},
			},
		}
		fakeClient := testclienthelper.NewAROFakeClientBuilder(cluster).Build()

		updateCalls := 0
		wrapper := &testClient{
			Client: fakeClient,
			updateFunc: func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
				updateCalls++
				if updateCalls == 1 {
					return kerrors.NewConflict(
						schema.GroupResource{Group: "config.openshift.io", Resource: "clusteroperators"},
						obj.GetName(),
						fmt.Errorf("the object has been modified"),
					)
				}
				return fakeClient.Update(ctx, obj, opts...)
			},
		}

		r := NewReconciler(log, wrapper)
		_, err := r.Reconcile(context.Background(), ctrl.Request{})
		utilerror.AssertErrorMessage(t, err, "")

		// Verify the ClusterOperator was created with correct owner reference and status
		co := &configv1.ClusterOperator{}
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: clusterOperatorName}, co); err != nil {
			t.Fatalf("failed to get ClusterOperator: %v", err)
		}

		if len(co.OwnerReferences) == 0 {
			t.Fatal("expected owner reference to be set on ClusterOperator")
		}
		if co.OwnerReferences[0].Name != arov1alpha1.SingletonClusterName {
			t.Errorf("expected owner reference name %q, got %q", arov1alpha1.SingletonClusterName, co.OwnerReferences[0].Name)
		}

		wantVersion := []configv1.OperandVersion{{Name: "operator", Version: version.GitCommit}}
		if diff := cmp.Diff(wantVersion, co.Status.Versions); diff != "" {
			t.Errorf("unexpected operator versions: %s", diff)
		}

		if updateCalls < 2 {
			t.Errorf("expected at least 2 update calls (1 conflict + 1 success), got %d", updateCalls)
		}
	})

	t.Run("conflict during Status().Update retries successfully", func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		cluster := &arov1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
			Status: arov1alpha1.ClusterStatus{
				Conditions: []operatorv1.OperatorCondition{
					utilconditions.ControllerDefaultAvailable("TestController"),
				},
			},
		}
		fakeClient := testclienthelper.NewAROFakeClientBuilder(cluster).Build()

		statusUpdateCalls := 0
		realStatus := fakeClient.Status()
		sw := &testStatusWriter{
			StatusWriter: realStatus,
			updateFunc: func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				statusUpdateCalls++
				// Call 1: getOrCreateClusterOperator sets initial status → let through
				// Call 2: setClusterOperatorStatus → return conflict
				// Call 3+: retry's setClusterOperatorStatus → let through
				if statusUpdateCalls == 2 {
					return kerrors.NewConflict(
						schema.GroupResource{Group: "config.openshift.io", Resource: "clusteroperators"},
						obj.GetName(),
						fmt.Errorf("the object has been modified"),
					)
				}
				return realStatus.Update(ctx, obj, opts...)
			},
		}

		wrapper := &testClient{
			Client:       fakeClient,
			statusWriter: sw,
		}

		r := NewReconciler(log, wrapper)
		_, err := r.Reconcile(context.Background(), ctrl.Request{})
		utilerror.AssertErrorMessage(t, err, "")

		co := &configv1.ClusterOperator{}
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: clusterOperatorName}, co); err != nil {
			t.Fatalf("failed to get ClusterOperator: %v", err)
		}

		// Verify status was eventually written correctly
		wantVersion := []configv1.OperandVersion{{Name: "operator", Version: version.GitCommit}}
		if diff := cmp.Diff(wantVersion, co.Status.Versions); diff != "" {
			t.Errorf("unexpected operator versions: %s", diff)
		}

		// Verify conditions were set (Available, Progressing, Degraded)
		if len(co.Status.Conditions) == 0 {
			t.Error("expected status conditions to be set")
		}

		// Verify there were retries
		if statusUpdateCalls < 3 {
			t.Errorf("expected at least 3 status update calls (1 init + 1 conflict + 1 success), got %d", statusUpdateCalls)
		}
	})
}
