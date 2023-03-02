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

	for _, cond := range cluster.Status.Conditions {
		switch cond.Type {
		case conditionName(controllerName, operatorv1.OperatorStatusTypeAvailable):
			conditions.Available = &cond
		case conditionName(controllerName, operatorv1.OperatorStatusTypeProgressing):
			conditions.Progressing = &cond
		case conditionName(controllerName, operatorv1.OperatorStatusTypeDegraded):
			conditions.Degraded = &cond
		}
	}

	return conditions, nil
}

func SetControllerConditions(ctx context.Context, c client.Client, conditions ControllerConditions) error {
	cluster, err := getCluster(ctx, c)
	if err != nil {
		return err
	}

	newConditions := cluster.Status.DeepCopy().Conditions
	v1helpers.SetOperatorCondition(&newConditions, *conditions.Available)
	v1helpers.SetOperatorCondition(&newConditions, *conditions.Progressing)
	v1helpers.SetOperatorCondition(&newConditions, *conditions.Degraded)

	if equality.Semantic.DeepEqual(cluster.Status.Conditions, newConditions) {
		return nil
	}

	cluster.Status.Conditions = newConditions
	if err := c.Status().Update(ctx, cluster); err != nil {
		return fmt.Errorf("error updating controller conditions: %w", err)
	}
	return nil
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
