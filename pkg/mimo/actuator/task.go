package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
)

type TaskHandler interface {
	OpenShiftDatabase() database.OpenShiftClusters
	Environment() env.Interface
}

type th struct {
	db  database.OpenShiftClusters
	env env.Interface
}

func (t *th) OpenShiftDatabase() database.OpenShiftClusters {
	return t.db
}

func (t *th) Environment() env.Interface {
	return t.env
}

type TaskFunc func(context.Context, TaskHandler, *api.OpenShiftClusterDocument, *api.MaintenanceManifestDocument) (api.MaintenanceManifestState, string)
