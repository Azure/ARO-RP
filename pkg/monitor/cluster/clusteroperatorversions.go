package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (mon *Monitor) emitClusterOperatorVersions(ctx context.Context) error {
	cv, err := mon.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		return err
	}

	cos, err := mon.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, co := range cos.Items {
		for _, v := range co.Status.Versions {
			if v.Name != "operator" {
				continue
			}

			if v.Version == desiredVersion(cv) {
				continue
			}

			mon.emitGauge("clusteroperator.versions", 1, map[string]string{
				"name":    co.Name,
				"version": v.Version,
			})
		}
	}

	return nil
}
