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

type conditionMetric struct {
	expectedStatus operatorv1.ConditionStatus
	metricName     string
}

var clusterObjectConditionMetrics = map[string]conditionMetric{
	arov1alpha1.InternetReachableFromMaster: {
		expectedStatus: operatorv1.ConditionTrue,
		metricName:     "arooperator.conditions",
	},
	arov1alpha1.InternetReachableFromWorker: {
		expectedStatus: operatorv1.ConditionTrue,
		metricName:     "arooperator.conditions",
	},
	arov1alpha1.ServicePrincipalValid: {
		expectedStatus: operatorv1.ConditionTrue,
		metricName:     "serviceprincipal.conditions",
	},
}

func (mon *Monitor) emitClusterObjectConditions(ctx context.Context) error {
	cluster, err := mon.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, c := range cluster.Status.Conditions {
		if status, ok := clusterObjectConditionMetrics[c.Type]; !ok || status.expectedStatus == c.Status {
			// Ignore conditions not in the map and ignore conditions with status True
			continue
		}

		mon.emitGauge(clusterObjectConditionMetrics[c.Type].metricName, 1, map[string]string{
			"status": string(c.Status),
			"type":   c.Type,
		})

		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"metric":  clusterObjectConditionMetrics[c.Type].metricName,
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}
