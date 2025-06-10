package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterSync(ctx context.Context) error {
	mon.log.Infof("mon.hourlyRun %t:", mon.hourlyRun)

	clusterSync, err := mon.hiveClusterManager.GetClusterSync(ctx, mon.doc)
	if err != nil {
		mon.log.Errorf("Error in getting the clustersync data %v", err)
		return err
	}
	if clusterSync == nil {
		mon.log.Info("Clustersync is NIL")
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
	mon.log.Info("Syncsets and SelectorSyncets both are NIL")
	return nil
}

func timeToString(t *metav1.Time) string {
	if t == nil {
		return ""
	}
	return t.String()
}
