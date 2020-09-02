package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
)

func (mon *Monitor) emitClusterVersions(ctx context.Context) error {
	cv, err := mon.getClusterVersion()
	if err != nil {
		return err
	}
	mon.emitGauge("cluster.versions", 1, map[string]string{
		"actualVersion":           actualVersion(cv),
		"desiredVersion":          desiredVersion(cv),
		"resourceProviderVersion": mon.oc.Properties.ProvisionedBy,
	})

	return nil
}

// actualVersion finds the actual current cluster state. The history is ordered by most
// recent first, so find the latest "Completed" status to get current
// cluster version
func actualVersion(cv *configv1.ClusterVersion) string {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return history.Version
		}
	}
	return ""
}

func desiredVersion(cv *configv1.ClusterVersion) string {
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		return cv.Spec.DesiredUpdate.Version
	}

	return cv.Status.Desired.Version
}
