package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

var aroOperatorConditionsExpected = map[status.ConditionType]corev1.ConditionStatus{
	arov1alpha1.InternetReachableFromMaster: corev1.ConditionTrue,
	arov1alpha1.InternetReachableFromWorker: corev1.ConditionTrue,
}

func (mon *Monitor) emitAroOperatorConditions(ctx context.Context) error {
	cluster, err := mon.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for _, c := range cluster.Status.Conditions {
		if aroOperatorConditionsExpected[c.Type] == c.Status {
			continue
		}

		mon.emitGauge("arooperator.conditions", 1, map[string]string{
			"status": string(c.Status),
			"type":   string(c.Type),
		})

		if mon.hourlyRun && c.Status == corev1.ConditionFalse {
			mon.log.WithFields(logrus.Fields{
				"metric":  "arooperator.conditions",
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}
