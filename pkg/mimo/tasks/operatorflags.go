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

func runSetGenevaLoggingOTelProfileTask(t mimo.TaskContext, profile string) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),
		steps.Action(func(ctx context.Context) error {
			return setGenevaLoggingOTelProfileInClusterDoc(ctx, profile)
		}),
		steps.Action(cluster.UpdateClusterOperatorFlags),
	}

	return run(t, s)
}

// Set the Operator flags for Geneva Logging to use OTel with the minimal-logs
// profile and update this in the cluster.
func SetOperatorFlagGenevaLoggingUseOTel(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	return runSetGenevaLoggingOTelProfileTask(t, operator.GenevaLoggingOTelProfileMinimalLogs)
}

// Set Geneva Logging to OTel with max-logs profile for global, master, and worker.
func SetOperatorFlagGenevaLoggingOTelProfileMaxLogs(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	return runSetGenevaLoggingOTelProfileTask(t, operator.GenevaLoggingOTelProfileMaxLogs)
}

// Set Geneva Logging to OTel with reduced-logs profile for global, master, and worker.
func SetOperatorFlagGenevaLoggingOTelProfileReducedLogs(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	return runSetGenevaLoggingOTelProfileTask(t, operator.GenevaLoggingOTelProfileReducedLogs)
}

// Set Geneva Logging to OTel with minimal-logs profile for global, master, and worker.
func SetOperatorFlagGenevaLoggingOTelProfileMinimalLogs(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	return runSetGenevaLoggingOTelProfileTask(t, operator.GenevaLoggingOTelProfileMinimalLogs)
}

func setGenevaLoggingOTelProfileInClusterDoc(ctx context.Context, profile string) error {
	if err := cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingEnabled, operator.FlagTrue); err != nil {
		return err
	}

	if err := cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingOTelProfile, profile); err != nil {
		return err
	}

	if err := cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingOTelMasterProfile, profile); err != nil {
		return err
	}

	return cluster.SetOperatorFlagInClusterDoc(ctx, operator.GenevaLoggingOTelWorkerProfile, profile)
}
