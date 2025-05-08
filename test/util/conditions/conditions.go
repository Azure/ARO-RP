package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

// AssertControllerConditions asserts that the ARO cluster resource contains the conditions expected in wantConditions.
func AssertControllerConditions(t *testing.T, ctx context.Context, client client.Client, wantConditions []operatorv1.OperatorCondition) {
	t.Helper()
	if len(wantConditions) == 0 {
		return
	}

	cluster := &arov1alpha1.Cluster{}
	if err := client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster); err != nil {
		t.Fatal(err)
	}
	for _, wantCondition := range wantConditions {
		gotCondition, err := findCondition(cluster.Status.Conditions, wantCondition.Type)
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(&wantCondition, gotCondition, cmpopts.EquateApproxTime(time.Second)); diff != "" {
			t.Error(diff)
		}
	}
}

func ControllerDefaultAvailable(name string) operatorv1.OperatorCondition {
	return operatorv1.OperatorCondition{
		Type:               fmt.Sprintf("%sController%s", name, operatorv1.OperatorStatusTypeAvailable),
		Status:             operatorv1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

func ControllerDefaultProgressing(name string) operatorv1.OperatorCondition {
	return operatorv1.OperatorCondition{
		Type:               fmt.Sprintf("%sController%s", name, operatorv1.OperatorStatusTypeProgressing),
		Status:             operatorv1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

func ControllerDefaultDegraded(name string) operatorv1.OperatorCondition {
	return operatorv1.OperatorCondition{
		Type:               fmt.Sprintf("%sController%s", name, operatorv1.OperatorStatusTypeDegraded),
		Status:             operatorv1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

func findCondition(conditions []operatorv1.OperatorCondition, conditionType string) (*operatorv1.OperatorCondition, error) {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			return &cond, nil
		}
	}

	return nil, fmt.Errorf("condition %s not found", conditionType)
}
