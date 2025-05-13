package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

const (
	authenticationTypeMetricsTopic = "cluster.AuthenticationType"
)

func (mon *Monitor) emitClusterAuthenticationType(ctx context.Context) error {
	if mon.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		mon.emitGauge(authenticationTypeMetricsTopic, 1, map[string]string{
			"type": "managedIdentity",
		})
	} else {
		mon.emitGauge(authenticationTypeMetricsTopic, 1, map[string]string{
			"type": "servicePrincipal",
		})
	}
	return nil
}
