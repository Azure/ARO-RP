package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

type TaskHandler interface {
	Environment() env.Interface
}

type th struct {
	env env.Interface
}

func (t *th) Environment() env.Interface {
	return t.env
}

type TaskFunc func(context.Context, TaskHandler, *api.MaintenanceManifestDocument, *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string)
