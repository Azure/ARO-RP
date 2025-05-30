package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	utilconditions "github.com/Azure/ARO-RP/test/util/conditions"
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
			cluster := &arov1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: arov1alpha1.SingletonClusterName,
				},
				Status: arov1alpha1.ClusterStatus{
					Conditions:      tt.start,
					OperatorVersion: "unknown",
				},
			}
			client := ctrlfake.NewClientBuilder().WithObjects(cluster).WithStatusSubresource(cluster).Build()

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
