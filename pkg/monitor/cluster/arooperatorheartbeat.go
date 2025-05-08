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

	aroDeployments, err := mon.listARODeployments(ctx)
	if err != nil {
		return err
	}

	for _, d := range aroDeployments.Items {
		_, present := aroOperatorDeploymentsReady[d.Name]
		if present {
			deploymentIsReady := ready.DeploymentIsReady(&d)
			mon.log.Infof("deployment %q is ready: %v, it's status: %+v", d.Name, deploymentIsReady, d.Status)
			aroOperatorDeploymentsReady[d.Name] = deploymentIsReady
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
