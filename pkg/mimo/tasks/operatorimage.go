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

// AutoUpdateOperatorImage updates the ARO operator to the image selected by the
// RP environment, matching the existing admin update operator path.
func AutoUpdateOperatorImage(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),
		steps.Action(cluster.UpdateAROOperatorImage),
		steps.Condition(cluster.AROOperatorDeploymentReady, 20*time.Minute, true),
		steps.Condition(cluster.EnsureAROOperatorRunningDesiredVersion, 5*time.Minute, true),
		steps.Action(cluster.SyncClusterObject),
	}

	return run(t, s)
}
