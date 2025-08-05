package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
)

func (mon *Monitor) emitClusterOperatorVersions(ctx context.Context) error {
	desiredVersion := ""
	if mon.clusterDesiredVersion != nil {
		desiredVersion = mon.clusterDesiredVersion.String()
	}

	var cont string
	l := &configv1.ClusterOperatorList{}

	for {
		err := mon.ocpclientset.List(ctx, l, client.Continue(cont), client.Limit(mon.queryLimit))
		if err != nil {
			return fmt.Errorf("error in ClusterOperator list operation: %w", err)
		}

		for _, co := range l.Items {
			for _, v := range co.Status.Versions {
				if v.Name != "operator" {
					continue
				}

				if v.Version == desiredVersion {
					continue
				}

				mon.emitGauge("clusteroperator.versions", 1, map[string]string{
					"name":    co.Name,
					"version": v.Version,
				})
			}
		}

		cont = l.Continue
		if cont == "" {
			break
		}
	}

	return nil
}
