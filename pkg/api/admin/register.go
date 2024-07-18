package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// APIVersion contains a version string as it will be used by clients
const APIVersion = "admin"

func init() {
	api.APIs[APIVersion] = &api.Version{
		OpenShiftClusterConverter:                      openShiftClusterConverter{},
		OpenShiftClusterStaticValidator:                openShiftClusterStaticValidator{},
		OpenShiftVersionConverter:                      openShiftVersionConverter{},
		OpenShiftVersionStaticValidator:                openShiftVersionStaticValidator{},
		PlatformWorkloadIdentityRoleSetConverter:       platformWorkloadIdentityRoleSetConverter{},
		PlatformWorkloadIdentityRoleSetStaticValidator: platformWorkloadIdentityRoleSetStaticValidator{},
		MaintenanceManifestConverter:                   maintenanceManifestConverter{},
	}
}
