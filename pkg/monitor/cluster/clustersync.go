package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterSync(ctx context.Context) error {
	clusterSync, err := mon.hiveClusterManager.GetClusterSync(ctx, mon.doc)
	if err != nil {
		return err
	}
	if clusterSync != nil {
		if clusterSync.Status.SyncSets != nil {
			for _, s := range clusterSync.Status.SyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"metric": "SyncSets",
					"name":   s.Name,
					"result": string(s.Result),
				})

				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"metric":             "SyncSets",
						"name":               s.Name,
						"result":             string(s.Result),
						"firstSuccessTime":   timeToString(s.FirstSuccessTime),
						"lastTransitionTime": timeToString(&s.LastTransitionTime),
						"failureMessage":     s.FailureMessage,
					}).Print()
				}
			}
		}
		if clusterSync.Status.SelectorSyncSets != nil {
			for _, s := range clusterSync.Status.SelectorSyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"metric": "SelectorSyncSets",
					"name":   s.Name,
					"result": string(s.Result),
				})
				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"metric":             "SelectorSyncSets",
						"name":               s.Name,
						"result":             string(s.Result),
						"firstSuccessTime":   timeToString(s.FirstSuccessTime),
						"lastTransitionTime": timeToString(&s.LastTransitionTime),
						"failureMessage":     s.FailureMessage,
					}).Print()
				}
			}
		}
	}
	return nil
}

func timeToString(t *metav1.Time) string {
	if t == nil {
		return ""
	}
	return t.String()
}
