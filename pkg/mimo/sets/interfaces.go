package sets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

type MaintenanceSet func(mimo.TaskContext, *api.MaintenanceManifestDocument, *api.OpenShiftClusterDocument) (api.MaintenanceManifestState, string)
