package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
)

var clusterDeploymentConditionsExpected = map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
	hivev1.ClusterReadyCondition: corev1.ConditionTrue,
	hivev1.UnreachableCondition:  corev1.ConditionFalse,
}

func (mon *Monitor) emitHiveRegistrationStatus(ctx context.Context) error {
	if mon.hiveClusterManager == nil {
		// TODO(hive): remove this if once we have Hive everywhere
		mon.log.Info("skipping: no hive cluster manager")
		return nil
	}

	if mon.oc.Properties.HiveProfile.Namespace == "" {
		mon.emitGauge("hive.condition.NoNamespace", 1, map[string]string{
			"reason": "NoNamespaceInClusterDocument",
		})
		return nil
	}

	cd, err := mon.hiveClusterManager.ClusterDeployment(ctx, mon.oc.Properties.HiveProfile.Namespace)
	if err != nil {
		return err
	}

	for _, condition := range cd.Status.Conditions {
		if expectedState, ok := clusterDeploymentConditionsExpected[condition.Type]; ok {
			if condition.Status != expectedState {
				name := fmt.Sprintf("hive.condition.%s", condition.Type)
				mon.emitGauge(name, 1, map[string]string{
					"reason": condition.Reason,
				})
			}
		}
	}
	return nil
}
