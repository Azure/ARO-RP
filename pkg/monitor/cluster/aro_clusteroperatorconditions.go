package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aro "github.com/Azure/ARO-RP/operator/apis/aro.openshift.io/v1alpha1"
)

var aroOperatorConditionsExpected = map[status.ConditionType]corev1.ConditionStatus{
	aro.InternetReachableFromMaster: corev1.ConditionTrue,
	aro.InternetReachableFromWorker: corev1.ConditionTrue,
}

func (mon *Monitor) emitAroOperatorConditions(ctx context.Context) error {
	cluster, err := mon.arocli.Clusters().Get(aro.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	mon.emitGauge("arooperator.count", int64(len(cluster.Status.Conditions)), nil)

	for _, c := range cluster.Status.Conditions {
		if aroOperatorConditionIsExpected(&c) {
			continue
		}

		mon.emitGauge("arooperator.conditions", 1, map[string]string{
			"name":   string(c.Type),
			"status": string(c.Status),
			"type":   string(c.Type),
		})

		if mon.logMessages {
			mon.log.WithFields(logrus.Fields{
				"metric":  "arooperator.conditions",
				"name":    c.Type,
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}

func aroOperatorConditionIsExpected(c *status.Condition) bool {
	return c.Status == aroOperatorConditionsExpected[c.Type]
}
