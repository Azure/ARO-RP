package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterOperatorConditionsIgnoreStruct struct {
	Name   string
	Type   configv1.ClusterStatusConditionType
	Status configv1.ConditionStatus
}

// clusterOperatorConditionsIgnore contains list of failures we know we can
// ignore for now
var clusterOperatorConditionsIgnore = map[clusterOperatorConditionsIgnoreStruct]struct{}{
	{"insights", "Disabled", configv1.ConditionFalse}:                                         {}, //not working on ARO
	{"insights", "Disabled", configv1.ConditionTrue}:                                          {}, //not working on ARO
	{"openshift-controller-manager", configv1.OperatorUpgradeable, configv1.ConditionUnknown}: {}, //inconsistent state. Operator is healthy
	{"service-ca", configv1.OperatorUpgradeable, configv1.ConditionUnknown}:                   {}, //inconsistent state. Operator is healthy
	{"service-catalog-apiserver", configv1.OperatorUpgradeable, configv1.ConditionUnknown}:    {}, //inconsistent state. Operator is healthy
}

var clusterOperatorConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorUpgradeable: configv1.ConditionTrue,
}

func (mon *Monitor) emitClusterOperatorsConditions(ctx context.Context) error {
	cos, err := mon.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, co := range cos.Items {
		sort.Slice(co.Status.Conditions, func(i, j int) bool { return co.Status.Conditions[i].Type < co.Status.Conditions[j].Type })
		for _, c := range co.Status.Conditions {
			if clusterOperatorConditionIsExpected(&co, &c) {
				continue
			}

			mon.emitGauge("clusteroperators.conditions", 1, map[string]string{
				"name":   co.Name,
				"type":   string(c.Type),
				"status": string(c.Status),
			})
		}
	}

	return nil
}

func clusterOperatorConditionIsExpected(co *configv1.ClusterOperator, c *configv1.ClusterOperatorStatusCondition) bool {
	if _, ok := clusterOperatorConditionsIgnore[clusterOperatorConditionsIgnoreStruct{
		Name:   co.Name,
		Type:   c.Type,
		Status: c.Status,
	}]; ok {
		return true
	}

	return c.Status == clusterOperatorConditionsExpected[c.Type]
}
