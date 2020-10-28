package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
)

var clusterVersionConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
	configv1.OperatorUpgradeable: configv1.ConditionTrue,
}

func (mon *Monitor) emitClusterVersionConditions(ctx context.Context) error {
	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return err
	}

	for _, c := range cv.Status.Conditions {
		if c.Status == clusterVersionConditionsExpected[c.Type] {
			continue
		}
		mon.emitGauge("clusterversion.conditions", 1, map[string]string{
			"status": string(c.Status),
			"type":   string(c.Type),
		})

		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"metric":  "clusterversion.conditions",
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}
