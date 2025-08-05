package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	configv1 "github.com/openshift/api/config/v1"
)

var clusterVersionConditionsExpected = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
	configv1.OperatorUpgradeable: configv1.ConditionTrue,
}

func (mon *Monitor) emitClusterVersionConditions(ctx context.Context) error {
	cv := &configv1.ClusterVersion{}
	err := mon.ocpclientset.Get(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return fmt.Errorf("failure fetching ClusterVersion: %w", err)
	}

	for _, c := range cv.Status.Conditions {
		if c.Status == clusterVersionConditionsExpected[c.Type] {
			continue
		}
		mon.emitGauge("clusterversion.conditions", 1, map[string]string{
			"status": string(c.Status),
			"type":   string(c.Type),
		})
	}

	return nil
}
