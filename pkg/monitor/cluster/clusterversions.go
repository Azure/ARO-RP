package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterVersions(ctx context.Context) error {
	cv, err := mon.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		return err
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

	var rpVersion string
	if mon.oc.Properties.ProvisionedBy != "" {
		rpVersion = mon.oc.Properties.ProvisionedBy
	} else {
		rpVersion = "unknown"
	}

	mon.emitGauge("cluster.versions", 1, map[string]string{
		"actualVersion":           actualVersion,
		"desiredVersion":          desiredVersion(cv),
		"resourceProviderVersion": rpVersion,
	})

	return nil
}

func desiredVersion(cv *configv1.ClusterVersion) string {
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		return cv.Spec.DesiredUpdate.Version
	}

	return cv.Status.Desired.Version
}
