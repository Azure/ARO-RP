package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clusterOperatorsConditionsWhitelist = map[configv1.ClusterStatusConditionType]struct{}{
	configv1.OperatorAvailable:   {},
	configv1.OperatorDegraded:    {},
	configv1.OperatorProgressing: {},
	configv1.OperatorUpgradeable: {},
}

var clusterOperatorsNotConditions = map[configv1.ClusterStatusConditionType]struct{}{
	configv1.OperatorAvailable:   {},
	configv1.OperatorUpgradeable: {},
}

func (mon *Monitor) emitClusterOperatorsMetrics(ctx context.Context) error {
	cos, err := mon.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, co := range cos.Items {
		for _, c := range co.Status.Conditions {
			if _, ok := clusterOperatorsConditionsWhitelist[c.Type]; !ok {
				continue
			}

			if _, ok := clusterOperatorsNotConditions[c.Type]; ok {
				if c.Status != configv1.ConditionTrue {
					mon.emitGauge("clusteroperators.conditions.count", 1, map[string]string{
						"clusteroperator": co.Name,
						"condition":       "Not" + string(c.Type),
					})
				}
			} else {
				if c.Status != configv1.ConditionFalse {
					mon.emitGauge("clusteroperators.conditions.count", 1, map[string]string{
						"clusteroperator": co.Name,
						"condition":       string(c.Type),
					})
				}
			}
		}

	out:
		for _, v := range co.Status.Versions {
			if v.Name == "operator" {
				mon.emitGauge("clusteroperators.version", 1, map[string]string{
					"clusteroperator": co.Name,
					"version":         v.Version,
				})
				break out
			}
		}
	}

	return nil
}
