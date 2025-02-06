package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	configv1 "github.com/openshift/api/config/v1"
)

type clusterOperatorConditionsIgnoreStruct struct {
	Name   string
	Type   configv1.ClusterStatusConditionType
	Status configv1.ConditionStatus
}

// clusterOperatorConditionsIgnore contains list of failures we know we can
// ignore for now
var clusterOperatorConditionsIgnore = map[clusterOperatorConditionsIgnoreStruct]struct{}{
	{"insights", "Disabled", configv1.ConditionFalse}:                                                    {},
	{"insights", "Disabled", configv1.ConditionTrue}:                                                     {},
	{"openshift-controller-manager", configv1.OperatorUpgradeable, configv1.ConditionUnknown}:            {},
	{"service-ca", configv1.OperatorUpgradeable, configv1.ConditionUnknown}:                              {},
	{"service-catalog-apiserver", configv1.OperatorUpgradeable, configv1.ConditionUnknown}:               {},
	{"cloud-controller-manager", "TrustedCABundleControllerControllerDegraded", configv1.ConditionFalse}: {},
	{"cloud-controller-manager", "TrustedCABundleControllerControllerAvailable", configv1.ConditionTrue}: {},
	{"cloud-controller-manager", "CloudConfigControllerDegraded", configv1.ConditionFalse}:               {},
}

var clusterOperatorConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorUpgradeable: configv1.ConditionTrue,
}

func (mon *Monitor) emitClusterOperatorConditions(ctx context.Context) error {
	cos, err := mon.listClusterOperators(ctx)
	if err != nil {
		return err
	}
	mon.emitGauge("clusteroperator.count", int64(len(cos.Items)), nil)

	for _, co := range cos.Items {
		for _, c := range co.Status.Conditions {
			if clusterOperatorConditionIsExpected(&co, &c) {
				continue
			}

			mon.emitGauge("clusteroperator.conditions", 1, map[string]string{
				"name":   co.Name,
				"status": string(c.Status),
				"type":   string(c.Type),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":  "clusteroperator.conditions",
					"name":    co.Name,
					"status":  c.Status,
					"type":    c.Type,
					"message": c.Message,
				}).Print()
			}
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
