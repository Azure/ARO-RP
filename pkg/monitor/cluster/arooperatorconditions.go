package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

const (
	operatorConditionsMetricsTopic = "arooperator.conditions"
)

var aroOperatorConditionsExpected = map[string]operatorv1.ConditionStatus{
	arov1alpha1.InternetReachableFromMaster: operatorv1.ConditionTrue,
	arov1alpha1.InternetReachableFromWorker: operatorv1.ConditionTrue,
	arov1alpha1.ServicePrincipalValid:       operatorv1.ConditionTrue,
	arov1alpha1.DefaultIngressCertificate:   operatorv1.ConditionTrue,
	arov1alpha1.MachineValid:                operatorv1.ConditionTrue,
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

		// filter out expected conditions (available=true, progressing=false, degraded=false)
		if (strings.HasSuffix(c.Type, "ControllerAvailable") && c.Status == operatorv1.ConditionTrue) ||
			(strings.HasSuffix(c.Type, "ControllerProgressing") && c.Status == operatorv1.ConditionFalse) ||
			(strings.HasSuffix(c.Type, "ControllerDegraded") && c.Status == operatorv1.ConditionFalse) {
			continue
		}

		mon.emitGauge(operatorConditionsMetricsTopic, 1, map[string]string{
			"status": string(c.Status),
			"type":   c.Type,
		})

		if mon.hourlyRun && c.Status == operatorv1.ConditionFalse {
			mon.log.WithFields(logrus.Fields{
				"metric":  operatorConditionsMetricsTopic,
				"status":  c.Status,
				"type":    c.Type,
				"message": c.Message,
			}).Print()
		}
	}

	return nil
}
