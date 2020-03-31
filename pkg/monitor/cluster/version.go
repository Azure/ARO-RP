package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	state "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterVersion() {
	ver, err := mon.configCli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		mon.log.Error(err)
		return
	}

	// Find the actual current cluster state. The history is ordered by most recent first,
	// so find the latest "Completed" status to get current cluster version
	actualVer := ""
	for _, history := range ver.Status.History {
		if history.State == state.CompletedUpdate {
			actualVer = history.Version
			break
		}
	}

	mon.emitGauge("cluster.version", 1, map[string]string{
		"desiredVersion": ver.Status.Desired.Version,
		"actualVersion":  actualVer,
	})
}
