package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/steps/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func UpdateOperatorFlags(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),

		steps.Action(cluster.UpdateClusterOperatorFlags),
	}

	return run(t, s)
}
