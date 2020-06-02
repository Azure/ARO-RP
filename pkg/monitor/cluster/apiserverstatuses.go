package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var apiServerOperatorConditionsExpected = map[string]operatorv1.ConditionStatus{
	"Available": operatorv1.ConditionTrue,
	"Degraded":  operatorv1.ConditionFalse,
}

func (mon *Monitor) emitOpenshiftApiServerStatuses(ctx context.Context) error {
	ds, err := mon.cli.AppsV1().DaemonSets("openshift-apiserver").Get("apiserver", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Emit the number of openshift apiservers running on the cluster
	if ds.Status.NumberAvailable != ds.Status.DesiredNumberScheduled {
		mon.emitGauge("apiserver.statuses", 1, map[string]string{
			"available": fmt.Sprintf("%d", ds.Status.NumberAvailable),
			"desired":   fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled),
		})
	}

	return nil
}

func (mon *Monitor) emitKubeApiServerStatuses(ctx context.Context) error {
	apiserver, err := mon.operatorcli.OperatorV1().KubeAPIServers().Get("cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Emit the kubeapiserver conditions
	for _, c := range apiserver.Status.Conditions {
		// Find condition key in map, as some condition keys may be substrings
		var keyName string
		found := false
		for key := range apiServerOperatorConditionsExpected {
			if strings.Contains(c.Type, key) {
				keyName = key
				found = true
			}
		}

		if found && c.Status != apiServerOperatorConditionsExpected[keyName] {
			mon.emitGauge("apiserver.conditions", 1, map[string]string{
				"status": string(c.Status),
				"type":   c.Type,
			})

			if mon.logMessages {
				mon.log.WithFields(logrus.Fields{
					"metric":  "apiserver.conditions",
					"status":  c.Status,
					"type":    c.Type,
					"message": c.Message,
				}).Print()
			}
		}
	}

	return nil
}

func (mon *Monitor) emitKubeApiServerNodeRevisionStatuses(ctx context.Context) error {
	apiserver, err := mon.operatorcli.OperatorV1().KubeAPIServers().Get("cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Emit the node status revisions
	for _, status := range apiserver.Status.NodeStatuses {
		if status.CurrentRevision != status.TargetRevision {
			mon.emitGauge("apiserver.nodestatuses", 1, map[string]string{
				"name":    status.NodeName,
				"current": fmt.Sprintf("%d", status.CurrentRevision),
				"target":  fmt.Sprintf("%d", status.TargetRevision),
			})

			if mon.logMessages {
				mon.log.WithFields(logrus.Fields{
					"name":    status.NodeName,
					"current": fmt.Sprintf("%d", status.CurrentRevision),
					"target":  fmt.Sprintf("%d", status.TargetRevision),
				}).Print()
			}
		}
	}

	return nil
}
