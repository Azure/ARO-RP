package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/v1helpers"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type AROController struct {
	Log    *logrus.Entry
	Client client.Client
	Name   string
}

func (c *AROController) SetConditions(ctx context.Context, cnds ...*operatorv1.OperatorCondition) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster, err := c.GetCluster(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve ARO cluster resource: %w", err)
		}

		newConditions := cluster.Status.DeepCopy().Conditions
		for _, cnd := range cnds {
			v1helpers.SetOperatorCondition(&newConditions, *cnd)
		}

		if equality.Semantic.DeepEqual(cluster.Status.Conditions, newConditions) {
			return nil
		}

		cluster.Status.Conditions = newConditions
		return c.Client.Status().Update(ctx, cluster)
	})
}

func (c *AROController) SetProgressing(ctx context.Context, message string) error {
	cnd := c.defaultProgressing()
	cnd.Status = operatorv1.ConditionTrue
	cnd.Message = message

	return c.SetConditions(ctx, cnd)
}

func (c *AROController) ClearProgressing(ctx context.Context) error {
	return c.SetConditions(ctx, c.defaultProgressing())
}

func (c *AROController) SetDegraded(ctx context.Context, err error) error {
	cnd := c.defaultDegraded()
	cnd.Status = operatorv1.ConditionTrue
	cnd.Message = err.Error()

	return c.SetConditions(ctx, cnd)
}

func (c *AROController) ClearDegraded(ctx context.Context) error {
	return c.SetConditions(ctx, c.defaultDegraded())
}

func (c *AROController) ClearConditions(ctx context.Context) error {
	return c.SetConditions(ctx, c.defaultAvailable(), c.defaultProgressing(), c.defaultDegraded())
}

func (c *AROController) GetCluster(ctx context.Context) (*arov1alpha1.Cluster, error) {
	cluster := &arov1alpha1.Cluster{}
	err := c.Client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, cluster)

	return cluster, err
}

func (c *AROController) defaultAvailable() *operatorv1.OperatorCondition {
	return &operatorv1.OperatorCondition{
		Type:   c.conditionName(operatorv1.OperatorStatusTypeAvailable),
		Status: operatorv1.ConditionTrue,
	}
}

func (c *AROController) defaultProgressing() *operatorv1.OperatorCondition {
	return &operatorv1.OperatorCondition{
		Type:   c.conditionName(operatorv1.OperatorStatusTypeProgressing),
		Status: operatorv1.ConditionFalse,
	}
}

func (c *AROController) defaultDegraded() *operatorv1.OperatorCondition {
	return &operatorv1.OperatorCondition{
		Type:   c.conditionName(operatorv1.OperatorStatusTypeDegraded),
		Status: operatorv1.ConditionFalse,
	}
}

func (c *AROController) conditionName(conditionType string) string {
	return c.Name + "Controller" + conditionType
}

func (c *AROController) IsClusterUpgrading(ctx context.Context) (bool, error) {
	cv := &configv1.ClusterVersion{}
	err := c.Client.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		err = fmt.Errorf("error getting the ClusterVersion: %w", err)
		c.Log.Error(err)
		return false, err
	}

	return version.IsClusterUpgrading(cv), nil
}

func (c *AROController) AllowRebootCausingReconciliation(ctx context.Context, instance *arov1alpha1.Cluster) (bool, error) {
	// If reconciliation is forced, perform it
	if instance.Spec.OperatorFlags.GetSimpleBoolean(operator.ForceReconciliation) {
		c.Log.Debugf("allowing reconciliation of %s because reconciliation forced", c.Name)
		return true, nil
	}

	// Allow the reconciliation if the cluster is upgrading
	isClusterUpgrading, err := c.IsClusterUpgrading(ctx)
	if err != nil {
		return false, err
	}

	return isClusterUpgrading, nil
}
