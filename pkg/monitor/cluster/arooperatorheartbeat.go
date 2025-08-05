package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/ready"
)

func (mon *Monitor) emitAroOperatorHeartbeat(ctx context.Context) error {
	aroOperatorDeploymentsReady := map[string]bool{
		"aro-operator-master": false,
		"aro-operator-worker": false,
	}

	l := &appsv1.DeploymentList{}
	err := mon.ocpclientset.List(ctx, l, client.InNamespace(pkgoperator.Namespace))
	if err != nil {
		return fmt.Errorf("failed listing ARO Operator deployments: %w", err)
	}

	for _, d := range l.Items {
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
