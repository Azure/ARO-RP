package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterSync(ctx context.Context) error {
	if mon.hiveClusterManager == nil {
		// TODO(hive): remove this once we have Hive everywhere.
		mon.log.Info("skipping: no hive cluster manager")
		return nil
	}

	clusterSync, err := mon.hiveClusterManager.GetClusterSync(ctx, mon.doc)
	if err != nil {
		return err
	}
	if clusterSync == nil {
		return nil
	} else {
		if clusterSync.Status.SyncSets == nil {
			return nil
		} else {
			for _, s := range clusterSync.Status.SyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"syncType": "SyncSets",
					"name":     s.Name,
					"result":   string(s.Result),
				})

				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"syncType":           "SyncSets",
						"name":               s.Name,
						"result":             string(s.Result),
						"firstSuccessTime":   timeToString(s.FirstSuccessTime),
						"lastTransitionTime": timeToString(&s.LastTransitionTime),
						"failureMessage":     s.FailureMessage,
					}).Print()
				}
			}
		}
		if clusterSync.Status.SelectorSyncSets == nil {
			return nil
		} else {
			for _, s := range clusterSync.Status.SelectorSyncSets {
				mon.emitGauge("hive.clustersync", 1, map[string]string{
					"syncType": "SelectorSyncSets",
					"name":     s.Name,
					"result":   string(s.Result),
				})
				if mon.hourlyRun {
					mon.log.WithFields(logrus.Fields{
						"syncType":           "SelectorSyncSets",
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
