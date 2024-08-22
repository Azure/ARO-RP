package sets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func ACRTokenChecker(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
	s := []steps.Step{
		steps.Action(cluster.EnsureACRTokenIsValid),
	}

	return run(t, s)
}
