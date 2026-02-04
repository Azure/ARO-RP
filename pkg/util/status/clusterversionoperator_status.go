package status

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	configv1 "github.com/openshift/api/config/v1"
)

const OperatorFailing configv1.ClusterStatusConditionType = "Failing"

// TODO: this is duplicate from clusterversioncondition.go
var clusterVersionConditionsHealthy = map[configv1.ClusterStatusConditionType]configv1.ConditionStatus{
	configv1.OperatorAvailable:   configv1.ConditionTrue,
	configv1.OperatorProgressing: configv1.ConditionFalse,
	configv1.OperatorDegraded:    configv1.ConditionFalse,
	configv1.OperatorUpgradeable: configv1.ConditionTrue,
	// TODO: Once https://github.com/openshift/cluster-version-operator/issues/437 this can go away,
	//      TL;DR: openshift/api updated, openshift/cvo did not catch up, therefore the condition missalignment
	OperatorFailing: configv1.ConditionFalse,
}

// ClusterVersionOperatorIsHealthy iterates core condotions and returns true
// if operators is considered healthy.
func ClusterVersionOperatorIsHealthy(status configv1.ClusterVersionStatus) bool {
	healthy := true
	for _, c := range status.Conditions {
		expect, ok := clusterVersionConditionsHealthy[c.Type]
		if ok && c.Status != expect {
			healthy = false
		}
	}
	return healthy
}
