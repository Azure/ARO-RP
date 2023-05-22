package conditions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

type ControllerConditions struct {
	Available, Progressing, Degraded *operatorv1.OperatorCondition
}

func GetControllerConditions(ctx context.Context, c client.Client, controllerName string) (ControllerConditions, error) {
	conditions := ControllerConditions{
		Available: &operatorv1.OperatorCondition{
			Type:   conditionName(controllerName, operatorv1.OperatorStatusTypeAvailable),
			Status: operatorv1.ConditionTrue,
		},
		Progressing: &operatorv1.OperatorCondition{
			Type:   conditionName(controllerName, operatorv1.OperatorStatusTypeProgressing),
			Status: operatorv1.ConditionFalse,
		},
		Degraded: &operatorv1.OperatorCondition{
			Type:   conditionName(controllerName, operatorv1.OperatorStatusTypeDegraded),
			Status: operatorv1.ConditionFalse,
		},
	}

	cluster, err := getCluster(ctx, c)
	if err != nil {
		return conditions, err
	}

	for i, cond := range cluster.Status.Conditions {
		switch cond.Type {
		case conditionName(controllerName, operatorv1.OperatorStatusTypeAvailable):
			conditions.Available = &cluster.Status.Conditions[i]
		case conditionName(controllerName, operatorv1.OperatorStatusTypeProgressing):
			conditions.Progressing = &cluster.Status.Conditions[i]
		case conditionName(controllerName, operatorv1.OperatorStatusTypeDegraded):
			conditions.Degraded = &cluster.Status.Conditions[i]
		}
	}

	return conditions, nil
}

func SetControllerConditions(ctx context.Context, c client.Client, cnds ControllerConditions) error {
	cluster, err := getCluster(ctx, c)
	if err != nil {
		return err
	}

	newConditions := cluster.Status.DeepCopy().Conditions
	v1helpers.SetOperatorCondition(&newConditions, *cnds.Available)
	v1helpers.SetOperatorCondition(&newConditions, *cnds.Progressing)
	v1helpers.SetOperatorCondition(&newConditions, *cnds.Degraded)

	if equality.Semantic.DeepEqual(cluster.Status.Conditions, newConditions) {
		return nil
	}

	cluster.Status.Conditions = newConditions
	if err := c.Status().Update(ctx, cluster); err != nil {
		return fmt.Errorf("error updating controller conditions: %w", err)
	}
	return nil
}

func SetControllerDegraded(ctx context.Context, c client.Client, cnds ControllerConditions, err error) error {
	cnds.Degraded.Status = operatorv1.ConditionTrue
	cnds.Degraded.Message = err.Error()

	return SetControllerConditions(ctx, c, cnds)
}

func ClearControllerDegraded(ctx context.Context, c client.Client, cnds ControllerConditions) error {
	cnds.Degraded.Status = operatorv1.ConditionFalse
	cnds.Degraded.Message = ""

	return SetControllerConditions(ctx, c, cnds)
}

func ClearControllerConditions(ctx context.Context, c client.Client, cnds ControllerConditions) error {
	cnds.Available.Status = operatorv1.ConditionTrue
	cnds.Available.Message = ""
	cnds.Progressing.Status = operatorv1.ConditionFalse
	cnds.Progressing.Message = ""
	cnds.Degraded.Status = operatorv1.ConditionFalse
	cnds.Degraded.Message = ""

	return SetControllerConditions(ctx, c, cnds)
}

func getCluster(ctx context.Context, c client.Client) (*arov1alpha1.Cluster, error) {
	cluster := &arov1alpha1.Cluster{}

	err := c.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func conditionName(controllerName string, conditionType string) string {
	return controllerName + "Controller" + conditionType
}
