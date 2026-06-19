package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/steps/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

// AutoUpdateOperatorImage checks for a newer ARO operator image version in ACR
// and schedules an operator update by patching the cluster document.
func AutoUpdateOperatorImage(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureOperatorImageUpdateScheduled),
	}

	return run(t, s)
}

