package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

const controllerName = "Test"
const controllerEnabled = "aro.test.enabled"

type TestReconciler struct {
	AROController
	reconciled bool
}

func newTestReconciler(log *logrus.Entry, client client.Client) *TestReconciler {
	r := &TestReconciler{
		AROController: AROController{
			Log:         log.WithField("controller", controllerName),
			Client:      client,
			Name:        controllerName,
			EnabledFlag: controllerEnabled,
		},
	}
	r.Reconciler = r
	return r
}

func (c *TestReconciler) SetupWithManager(ctrl.Manager) error {
	return nil
}

func (c *TestReconciler) ReconcileEnabled(ctx context.Context, req ctrl.Request, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	c.reconciled = true
	c.SetProgressing(ctx, "")
	return ctrl.Result{}, nil
}

func condition(conditionType string, status operatorv1.ConditionStatus) operatorv1.OperatorCondition {
	return operatorv1.OperatorCondition{
		Type:               controllerName + "Controller" + conditionType,
		Status:             status,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

func TestReconcile(t *testing.T) {
	for _, tt := range []struct {
		name              string
		controllerEnabled bool
		startConditions   []operatorv1.OperatorCondition
		wantConditions    []operatorv1.OperatorCondition
	}{
		{
			name:              "enabled controller calls ReconcileEnabled",
			controllerEnabled: true,
			startConditions: []operatorv1.OperatorCondition{
				condition(
					operatorv1.OperatorStatusTypeAvailable,
					operatorv1.ConditionTrue),
				condition(
					operatorv1.OperatorStatusTypeProgressing,
					operatorv1.ConditionFalse),
				condition(
					operatorv1.OperatorStatusTypeDegraded,
					operatorv1.ConditionFalse),
			},
			wantConditions: []operatorv1.OperatorCondition{
				condition(
					operatorv1.OperatorStatusTypeAvailable,
					operatorv1.ConditionTrue),
				condition(
					operatorv1.OperatorStatusTypeProgressing,
					operatorv1.ConditionTrue),
				condition(
					operatorv1.OperatorStatusTypeDegraded,
					operatorv1.ConditionFalse),
			},
		},
		{
			name:              "disabled controller calls ReconcileDisabled",
			controllerEnabled: false,
			startConditions: []operatorv1.OperatorCondition{
				condition(
					operatorv1.OperatorStatusTypeAvailable,
					operatorv1.ConditionFalse),
				condition(
					operatorv1.OperatorStatusTypeProgressing,
					operatorv1.ConditionTrue),
				condition(
					operatorv1.OperatorStatusTypeDegraded,
					operatorv1.ConditionFalse),
			},
			wantConditions: []operatorv1.OperatorCondition{
				condition(
					operatorv1.OperatorStatusTypeAvailable,
					operatorv1.ConditionTrue),
				condition(
					operatorv1.OperatorStatusTypeProgressing,
					operatorv1.ConditionFalse),
				condition(
					operatorv1.OperatorStatusTypeDegraded,
					operatorv1.ConditionFalse),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger())

			client := ctrlfake.NewClientBuilder().
				WithObjects(
					&arov1alpha1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: arov1alpha1.SingletonClusterName,
						},
						Spec: arov1alpha1.ClusterSpec{
							OperatorFlags: arov1alpha1.OperatorFlags{
								controllerEnabled: strconv.FormatBool(tt.controllerEnabled),
							},
						},
						Status: arov1alpha1.ClusterStatus{
							Conditions: tt.startConditions,
						},
					},
				).Build()

			ctx := context.Background()
			r := newTestReconciler(log, client)
			_, err := r.Reconcile(ctx, ctrl.Request{})

			utilerror.AssertErrorMessage(t, err, "")
			utilconditions.AssertControllerConditions(t, ctx, r.Client, tt.wantConditions)
			if r.reconciled != tt.controllerEnabled {
				if tt.controllerEnabled {
					t.Errorf("enabled controller did not reconcile")
				} else {
					t.Errorf("disabled controller reconciled anyway")
				}
			}
		})
	}
}

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
		action func(c AROController)
		want   []operatorv1.OperatorCondition
	}{
		{
			name:  "SetConditions - sets all provided conditions",
			start: []operatorv1.OperatorCondition{internetReachable},
			action: func(c AROController) {
				c.SetConditions(ctx, &defaultAvailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, defaultAvailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetConditions - if condition exists and status matches, does not update",
			start: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast},
			action: func(c AROController) {
				c.SetConditions(ctx, &defaultAvailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetConditions - if condition exists and status does not match, updates",
			start: []operatorv1.OperatorCondition{internetReachable, defaultAvailableInPast},
			action: func(c AROController) {
				c.SetConditions(ctx, &unavailable, &defaultProgressing, &defaultDegraded)
			},
			want: []operatorv1.OperatorCondition{internetReachable, unavailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetProgressing - sets Progressing to true with message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
			action: func(c AROController) {
				c.SetProgressing(ctx, isProgressing.Message)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, isProgressing, defaultDegraded},
		},
		{
			name:  "ClearProgressing - sets Progressing to false and clears message",
			start: []operatorv1.OperatorCondition{defaultAvailable, isProgressing, defaultDegraded},
			action: func(c AROController) {
				c.ClearProgressing(ctx)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
		},
		{
			name:  "SetDegraded - sets Degraded to true with message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
			action: func(c AROController) {
				err := errors.New(isDegraded.Message)
				c.SetDegraded(ctx, err)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, isDegraded},
		},
		{
			name:  "ClearDegraded - sets Degraded to false and clears message",
			start: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, isDegraded},
			action: func(c AROController) {
				c.ClearDegraded(ctx)
			},
			want: []operatorv1.OperatorCondition{defaultAvailable, defaultProgressing, defaultDegraded},
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

			tt.action(controller)
			utilconditions.AssertControllerConditions(t, ctx, client, tt.want)
		})
	}
}
