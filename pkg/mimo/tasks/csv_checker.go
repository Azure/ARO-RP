package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/steps/cluster"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

// ConcerningCSVChecker detects Red Hat Operators that were inadvertently
// upgraded to 4.18 catalog versions on 4.12-4.17 clusters due to an
// incorrect catalog content release on 2026-02-03.
// https://access.redhat.com/solutions/7137887
func ConcerningCSVChecker(t mimo.TaskContext, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) error {
	s := []steps.Step{
		steps.Action(cluster.EnsureAPIServerIsUp),

		steps.Action(cluster.DetectConcerningClusterServiceVersions),
	}

	return run(t, s)
}
