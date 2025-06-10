package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterSync(ctx context.Context) error {
	if mon.hiveClusterManager == nil {
		// TODO(hive): remove this once we have HiveManager available everywhere
		mon.log.Info("skipping: no hive cluster manager")
		return nil
	}

	clusterSync, err := mon.hiveClusterManager.GetClusterSync(ctx, mon.doc)
	if err != nil {
		mon.log.Errorf("Error in getting the clustersync data: %v", err)
		return err
	}
	if clusterSync != nil {
		if clusterSync.Status.SyncSets != nil {
			for _, s := range clusterSync.Status.SyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"name":   s.Name,
					"status": string(s.Result),
					"type":   "SyncSets",
					"reason": cleanString(s.FailureMessage),
				})

				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"name":               s.Name,
						"status":             string(s.Result),
						"type":               "SyncSets",
						"firstSuccessTime":   timeToString(s.FirstSuccessTime),
						"lastTransitionTime": timeToString(&s.LastTransitionTime),
						"reason":             cleanString(s.FailureMessage),
					}).Print()
				}
			}
		}
		if clusterSync.Status.SelectorSyncSets != nil {
			for _, s := range clusterSync.Status.SelectorSyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"name":   s.Name,
					"status": string(s.Result),
					"type":   "SelectorSyncSets",
					"reason": cleanString(s.FailureMessage),
				})
				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"name":               s.Name,
						"status":             string(s.Result),
						"type":               "SelectorSyncSets",
						"firstSuccessTime":   timeToString(s.FirstSuccessTime),
						"lastTransitionTime": timeToString(&s.LastTransitionTime),
						"reason":             cleanString(s.FailureMessage),
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

func cleanString(s string) string {
	re, err := regexp.Compile(`[^\w\s\-_]|\n`)
	if err != nil {
		return s
	}
	return re.ReplaceAllString(s, "")
}
