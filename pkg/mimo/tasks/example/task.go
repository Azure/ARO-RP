package example

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
)

func ExampleTask(ctx context.Context, th tasks.TaskContext, manifest *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string) {
	ch, err := th.ClientHelper()
	if err != nil {
		return api.MaintenanceManifestStateFailed, err.Error()
	}

	cv := &configv1.ClusterVersion{}

	err = ch.GetOne(ctx, types.NamespacedName{Name: "version"}, cv)
	if err != nil {
		return api.MaintenanceManifestStateFailed, fmt.Errorf("unable to get ClusterVersion: %w", err).Error()
	}

	return api.MaintenanceManifestStateCompleted, fmt.Sprintf("cluster version is: %s", cv.Status.History[0].Version)
}
