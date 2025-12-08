package clusteroperators

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
)

func IsOperatorAvailable(operator *configv1.ClusterOperator) bool {
	m := make(map[configv1.ClusterStatusConditionType]configv1.ConditionStatus, len(operator.Status.Conditions))
	for _, cond := range operator.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[configv1.OperatorAvailable] == configv1.ConditionTrue && m[configv1.OperatorProgressing] == configv1.ConditionFalse && m[configv1.OperatorDegraded] == configv1.ConditionFalse
}

func OperatorStatusText(operator *configv1.ClusterOperator) string {
	m := make(map[configv1.ClusterStatusConditionType]configv1.ConditionStatus, len(operator.Status.Conditions))
	for _, cond := range operator.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return fmt.Sprintf("%s %s=%s, %s=%s", operator.Name,
		configv1.OperatorAvailable, m[configv1.OperatorAvailable], configv1.OperatorProgressing, m[configv1.OperatorProgressing],
	)
}
