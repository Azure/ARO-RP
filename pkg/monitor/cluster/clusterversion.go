package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterVersion(ctx context.Context) error {
	cv, err := mon.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		return err
	}

	desiredVersion := cv.Status.Desired.Version
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		desiredVersion = cv.Spec.DesiredUpdate.Version
	}

	// Find the actual current cluster state. The history is ordered by most
	// recent first, so find the latest "Completed" status to get current
	// cluster version
	var actualVersion string
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			actualVersion = history.Version
			break
		}
	}

	mon.emitGauge("cluster.version", 1, map[string]string{
		"actualVersion":  actualVersion,
		"desiredVersion": desiredVersion,
	})

	return nil
}
