package clusteroperators

import (
	configv1 "github.com/openshift/api/config/v1"
)

func IsOperatorAvailable(operator *configv1.ClusterOperator) bool {
	m := make(map[configv1.ClusterStatusConditionType]configv1.ConditionStatus, len(operator.Status.Conditions))
	for _, cond := range operator.Status.Conditions {
		m[cond.Type] = cond.Status
	}
	return m[configv1.OperatorAvailable] == configv1.ConditionTrue && m[configv1.OperatorProgressing] == configv1.ConditionFalse
}
