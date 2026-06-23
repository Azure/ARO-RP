package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/steps/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

// Update the Operator and the Cluster object (with associated feature flags).
func UpdateOperator(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.DeployOperatorIntoCluster),

		steps.Condition(cluster.WaitForAROOperatorReady, 20*time.Minute, true),
		steps.Condition(cluster.WaitForAROOperatorRunningDesiredVersion, 5*time.Minute, true),

		// Once the ARO Operator is updated, synchronize the Cluster object.
		// This is done after the ARO Operator is potentially updated so that
		// any flag changes that happen in the same request only apply on the
		// new Operator. Otherwise, it is possible for a flag change to occur on
		// the old Operator version, then require reconciling to a new version a
		// second time (e.g. DNSMasq changes) with the associated node cyclings
		// for the resource updates.
		steps.Action(cluster.SyncClusterObject),
	}

	return run(t, s)
}

// Reset the version of the Operator to the same version as the RP and updates
// it and the Cluster object.
func ResetOperatorVersion(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.ResetOperatorVersion),
		steps.Action(cluster.DeployOperatorIntoCluster),

		steps.Condition(cluster.WaitForAROOperatorReady, 20*time.Minute, true),
		steps.Condition(cluster.WaitForAROOperatorRunningDesiredVersion, 5*time.Minute, true),

		// See the comment for this function in UpdateOperator for why we sync
		// the Cluster object separately.
		steps.Action(cluster.SyncClusterObject),
	}

	return run(t, s)
}

// Sync the Cluster object into the Cluster.
func SyncClusterObject(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.SyncClusterObject),
	}

	return run(t, s)
}
