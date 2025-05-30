package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/hive"
)

var clusterDeploymentConditionsExpected = map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
	hivev1.ClusterReadyCondition: corev1.ConditionTrue,
	hivev1.UnreachableCondition:  corev1.ConditionFalse,
}

func (mon *Monitor) emitHiveRegistrationStatus(ctx context.Context) error {
	if mon.hiveclientset == nil {
		// TODO(hive): remove this once we have Hive everywhere
		mon.log.Info("skipping: no hive cluster manager")
		return nil
	}

	if mon.oc.Properties.HiveProfile.Namespace == "" {
		return fmt.Errorf("cluster %s not adopted. No namespace in the clusterdocument", mon.oc.Name)
	}

	cd, err := mon.retrieveClusterDeployment(ctx)
	if err != nil {
		return err
	}

	conditions := mon.filterClusterDeploymentConditions(ctx, cd, clusterDeploymentConditionsExpected)
	mon.emitFilteredClusterDeploymentMetrics(conditions)
	return nil
}

func (mon *Monitor) retrieveClusterDeployment(ctx context.Context) (*hivev1.ClusterDeployment, error) {
	cd := &hivev1.ClusterDeployment{}
	err := mon.hiveclientset.Get(ctx, client.ObjectKey{
		Namespace: mon.oc.Properties.HiveProfile.Namespace,
		Name:      hive.ClusterDeploymentName,
	}, cd)
	if err != nil {
		return nil, err
	}

	return cd, nil
}

func (mon *Monitor) filterClusterDeploymentConditions(ctx context.Context, cd *hivev1.ClusterDeployment, clusterDeploymentConditionsExpected map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus) []hivev1.ClusterDeploymentCondition {
	conditions := make([]hivev1.ClusterDeploymentCondition, 0)
	for _, condition := range cd.Status.Conditions {
		if expectedState, ok := clusterDeploymentConditionsExpected[condition.Type]; ok {
			if condition.Status != expectedState {
				conditions = append(conditions, condition)
			}
		}
	}

	return conditions
}

func (mon *Monitor) emitFilteredClusterDeploymentMetrics(conditions []hivev1.ClusterDeploymentCondition) {
	for _, condition := range conditions {
		mon.emitGauge("hive.clusterdeployment.conditions", 1, map[string]string{
			"type":   string(condition.Type),
			"reason": condition.Reason,
		})
	}
}
