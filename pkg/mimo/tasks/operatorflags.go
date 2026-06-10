package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/steps/cluster"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func UpdateOperatorFlags(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),

		steps.Action(cluster.UpdateClusterOperatorFlags),
	}

	return run(t, s)
}

// Set the Operator flag for Geneva Logging mode to use the OTel-based exporter
// and update this in the cluster.
func SetOperatorFlagGenevaLoggingUseOTel(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),
		steps.Action(func(ctx context.Context) error {
			return cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingMode, operator.GenevaLoggingModeOTel)
		}),
		steps.Action(cluster.UpdateClusterOperatorFlags),
	}

	return run(t, s)
}

// Set the Operator flag for Geneva Logging mode to use the fluentbit and
// MDSD-based forwarder and update this in the cluster.
func SetOperatorFlagGenevaLoggingUseMDSD(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),
		steps.Action(func(ctx context.Context) error {
			return cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingMode, operator.GenevaLoggingModeMDSD)
		}),
		steps.Action(cluster.UpdateClusterOperatorFlags),
	}

	return run(t, s)
}
