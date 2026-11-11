package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestConditions(t *testing.T) {
	ctx := context.Background()

	controllerName := "Fake"

	now := metav1.NewTime(time.Now())
	past := metav1.NewTime(now.Add(-1 * time.Hour))

	internetReachable := operatorv1.OperatorCondition{
		Type:               arov1alpha1.InternetReachableFromMaster,
		Status:             operatorv1.ConditionFalse,
		LastTransitionTime: now,
	}

	defaultAvailable := utilconditions.ControllerDefaultAvailable(controllerName)
	defaultProgressing := utilconditions.ControllerDefaultProgressing(controllerName)
	defaultDegraded := utilconditions.ControllerDefaultDegraded(controllerName)

	defaultAvailableInPast := *defaultAvailable.DeepCopy()
	defaultAvailableInPast.LastTransitionTime = past

	unavailable := *defaultAvailable.DeepCopy()
	unavailable.Status = operatorv1.ConditionFalse
	unavailable.Message = "Something bad happened"

	isProgressing := *defaultProgressing.DeepCopy()
	isProgressing.Status = operatorv1.ConditionTrue
	isProgressing.Message = "Controller is performing task"

	isDegraded := *defaultDegraded.DeepCopy()
	isDegraded.Status = operatorv1.ConditionTrue
	isDegraded.Message = "Controller failed to perform task"

	for _, tt := range []struct {
		name   string
		start  []operatorv1.OperatorCondition
		action func(c AROController) error
		want   []operatorv1.OperatorCondition
	}{
		{
			name:  "SetConditions - sets all provided conditions",
			start: []operatorv1.OperatorCondition{internetReachable},
			action: func(c AROController) error {
				return c.SetConditions(ctx, &defaultAvailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, defaultAvailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetConditions - if condition exists and status matches, does not update",
			start: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast},
			action: func(c AROController) error {
				return c.SetConditions(ctx, &defaultAvailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetConditions - if condition exists and status does not match, updates",
			start: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast},
			action: func(c AROController) error {
				return c.SetConditions(ctx, &unavailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, unavailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetProgressing - sets Progressing to true with message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
			action: func(c AROController) error {
				return c.SetProgressing(ctx, isProgressing.Message)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, isProgressing, defaultDegraded},
		},
		{
			name:  "ClearProgressing - sets Progressing to false and clears message",
			start: []operatorv1.OperatorCondition{defaultAvailable, isProgressing, defaultDegraded},
			action: func(c AROController) error {
				return c.ClearProgressing(ctx)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetDegraded - sets Degraded to true with message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
			action: func(c AROController) error {
				err := errors.New(isDegraded.Message)
				return c.SetDegraded(ctx, err)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, isDegraded},
		},
		{
			name:  "ClearDegraded - sets Degraded to false and clears message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, isDegraded},
			action: func(c AROController) error {
				return c.ClearDegraded(ctx)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, log := testlog.LogForTesting(t)
			client := testclienthelper.NewAROFakeClientBuilder(
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
				Log:    log,
				Client: client,
				Name:   controllerName,
			}

			if err := tt.action(controller); err != nil {
				t.Fatalf("unexpected error from action: %v", err)
			}
			utilconditions.AssertControllerConditions(t, ctx, client, tt.want)
		})
	}
}

// testStatusWriter wraps a StatusWriter to intercept Update calls for conflict injection.
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

// testClient wraps a client.Client and overrides Status() to return a custom StatusWriter.
type testClient struct {
	client.Client
	statusWriter client.StatusWriter
}

func (c *testClient) Status() client.StatusWriter {
	return c.statusWriter
}

func newConflictError(name string) error {
	return kerrors.NewConflict(
		schema.GroupResource{Group: "aro.openshift.io", Resource: "clusters"},
		name,
		fmt.Errorf("the object has been modified"),
	)
}

func TestSetConditionsRetryOnConflict(t *testing.T) {
	ctx := context.Background()
	controllerName := "Fake"

	t.Run("retry re-reads and preserves concurrent condition changes", func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		initialCondition := operatorv1.OperatorCondition{
			Type:   arov1alpha1.InternetReachableFromMaster,
			Status: operatorv1.ConditionFalse,
		}

		fakeClient := testclienthelper.NewAROFakeClientBuilder(
			&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Status: arov1alpha1.ClusterStatus{
					Conditions:      []operatorv1.OperatorCondition{initialCondition},
					OperatorVersion: "unknown",
				},
			},
		).Build()

		concurrentCondition := operatorv1.OperatorCondition{
			Type:   "AnotherControllerAvailable",
			Status: operatorv1.ConditionTrue,
		}

		calls := 0
		realStatus := fakeClient.Status()
		sw := &testStatusWriter{
			StatusWriter: realStatus,
			updateFunc: func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				calls++
				if calls == 1 {
					// Simulate another controller writing a condition concurrently
					cluster := &arov1alpha1.Cluster{}
					if err := fakeClient.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster); err != nil {
						t.Fatalf("onConflict Get failed: %v", err)
					}
					cluster.Status.Conditions = append(cluster.Status.Conditions, concurrentCondition)
					if err := realStatus.Update(ctx, cluster); err != nil {
						t.Fatalf("onConflict Status Update failed: %v", err)
					}
					return newConflictError(arov1alpha1.SingletonClusterName)
				}
				return realStatus.Update(ctx, obj, opts...)
			},
		}

		controller := AROController{
			Log:    log,
			Client: &testClient{Client: fakeClient, statusWriter: sw},
			Name:   controllerName,
		}

		available := utilconditions.ControllerDefaultAvailable(controllerName)
		if err := controller.SetConditions(ctx, &available); err != nil {
			t.Fatalf("SetConditions returned unexpected error: %v", err)
		}

		// Verify all three conditions are present: initial + concurrent + our new one
		cluster := &arov1alpha1.Cluster{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster); err != nil {
			t.Fatalf("failed to get cluster: %v", err)
		}

		found := map[string]bool{}
		for _, c := range cluster.Status.Conditions {
			found[c.Type] = true
		}
		for _, wantType := range []string{
			arov1alpha1.InternetReachableFromMaster,
			"AnotherControllerAvailable",
			controllerName + "Controller" + operatorv1.OperatorStatusTypeAvailable,
		} {
			if !found[wantType] {
				t.Errorf("condition %q not found in %v", wantType, cluster.Status.Conditions)
			}
		}

		if calls != 2 {
			t.Errorf("expected 2 status update calls (1 conflict + 1 success), got %d", calls)
		}
	})

	t.Run("exhausted retries return error", func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		fakeClient := testclienthelper.NewAROFakeClientBuilder(
			&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Status:     arov1alpha1.ClusterStatus{OperatorVersion: "unknown"},
			},
		).Build()

		sw := &testStatusWriter{
			StatusWriter: fakeClient.Status(),
			updateFunc: func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				return newConflictError(arov1alpha1.SingletonClusterName)
			},
		}

		controller := AROController{
			Log:    log,
			Client: &testClient{Client: fakeClient, statusWriter: sw},
			Name:   controllerName,
		}

		available := utilconditions.ControllerDefaultAvailable(controllerName)
		err := controller.SetConditions(ctx, &available)
		if err == nil {
			t.Fatal("expected error from exhausted retries, got nil")
		}
		if !kerrors.IsConflict(err) {
			t.Fatalf("expected conflict error, got: %v", err)
		}
	})

	t.Run("non-conflict errors are not retried", func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		fakeClient := testclienthelper.NewAROFakeClientBuilder(
			&arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: arov1alpha1.SingletonClusterName},
				Status:     arov1alpha1.ClusterStatus{OperatorVersion: "unknown"},
			},
		).Build()

		calls := 0
		sw := &testStatusWriter{
			StatusWriter: fakeClient.Status(),
			updateFunc: func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				calls++
				return fmt.Errorf("network unavailable")
			},
		}

		controller := AROController{
			Log:    log,
			Client: &testClient{Client: fakeClient, statusWriter: sw},
			Name:   controllerName,
		}

		available := utilconditions.ControllerDefaultAvailable(controllerName)
		err := controller.SetConditions(ctx, &available)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != 1 {
			t.Errorf("expected exactly 1 status update call (no retry for non-conflict), got %d", calls)
		}
	})
}
