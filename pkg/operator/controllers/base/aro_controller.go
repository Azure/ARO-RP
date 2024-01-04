package base

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type AROController struct {
	Log    *logrus.Entry
	Client client.Client
	Name   string
}

func (c *AROController) SetConditions(ctx context.Context, cnds ...*operatorv1.OperatorCondition) {
	cluster, err := c.GetCluster(ctx)
	if err != nil {
		c.Log.Warn("Failed to retrieve ARO cluster resource")
		return
	}

	newConditions := cluster.Status.DeepCopy().Conditions
	for _, cnd := range cnds {
		v1helpers.SetOperatorCondition(&newConditions, *cnd)
	}

	if equality.Semantic.DeepEqual(cluster.Status.Conditions, newConditions) {
		return
	}

	cluster.Status.Conditions = newConditions
	if err := c.Client.Status().Update(ctx, cluster); err != nil {
		c.Log.Error("error updating controller conditions", err)
	}
}

func (c *AROController) SetProgressing(ctx context.Context, message string) {
	cnd := c.defaultProgressing()
	cnd.Status = operatorv1.ConditionTrue
	cnd.Message = message

	c.SetConditions(ctx, cnd)
}

func (c *AROController) ClearProgressing(ctx context.Context) {
	c.SetConditions(ctx, c.defaultProgressing())
}

func (c *AROController) SetDegraded(ctx context.Context, err error) {
	cnd := c.defaultDegraded()
	cnd.Status = operatorv1.ConditionTrue
	cnd.Message = err.Error()

	c.SetConditions(ctx, cnd)
}

func (c *AROController) ClearDegraded(ctx context.Context) {
	c.SetConditions(ctx, c.defaultDegraded())
}

func (c *AROController) ClearConditions(ctx context.Context) {
	c.SetConditions(ctx, c.defaultAvailable(), c.defaultProgressing(), c.defaultDegraded())
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
		c.Log.Errorf("error getting the ClusterVersion: %v", err)
		return false, err
	}

	return version.IsClusterUpgrading(cv), nil
}
