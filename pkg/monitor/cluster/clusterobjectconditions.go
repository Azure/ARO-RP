package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

var clusterObjectConditionsExpected = map[string]operatorv1.ConditionStatus{
	// ARO Operator Conditions
	arov1alpha1.InternetReachableFromMaster: operatorv1.ConditionTrue,
	arov1alpha1.InternetReachableFromWorker: operatorv1.ConditionTrue,
	// Service Principal Condition
	arov1alpha1.ServicePrincipalValid: operatorv1.ConditionTrue,
}

var metricname = map[string]string{
	// ARO Operator Metrics
	arov1alpha1.InternetReachableFromMaster: "arooperator.conditions",
	arov1alpha1.InternetReachableFromWorker: "arooperator.conditions",
	// Service principal Metric
	arov1alpha1.ServicePrincipalValid: "serviceprincipal.conditions",
}

func (mon *Monitor) emitClusterObjectConditions(ctx context.Context) error {
	cluster, err := mon.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, c := range cluster.Status.Conditions {
		if _, ok := clusterObjectConditionsExpected[c.Type]; !ok {
			// Ignore conditions not in the map
			continue
		}
		if clusterObjectConditionsExpected[c.Type] == c.Status {
			continue
		}

		mon.emitGauge(metricname[c.Type], 1, map[string]string{
			"status": string(c.Status),
			"type":   c.Type,
		})

		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"metric":  metricname[c.Type],
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}
