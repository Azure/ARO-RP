package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/ready"
)

func (mon *Monitor) emitAroOperatorHeartbeat(ctx context.Context) error {
	aroOperatorDeploymentsReady := map[string]bool{
		"aro-operator-master": false,
		"aro-operator-worker": false}

	aroDeployments, err := mon.listDeployments()
	if err != nil {
		return err
	}

	for _, d := range aroDeployments.Items {
		if d.Namespace != "openshift-azure-operator" {
			continue
		}

		_, present := aroOperatorDeploymentsReady[d.Name]
		if present {
			aroOperatorDeploymentsReady[d.Name] = ready.DeploymentIsReady(&d)
		}
	}

	for n, r := range aroOperatorDeploymentsReady {
		value := int64(0)
		if r {
			value = 1
		}
		mon.emitGauge("arooperator.heartbeat", value, map[string]string{
			"name": n,
		})
	}
	return nil
}
