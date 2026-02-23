package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// APIVersion contains a version string as it will be used by clients
const APIVersion = "2025-07-25"

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

func register() {
	api.APIs[APIVersion] = &api.Version{
		OpenShiftClusterConverter:                openShiftClusterConverter{},
		OpenShiftClusterStaticValidator:          openShiftClusterStaticValidator{},
		OpenShiftClusterCredentialsConverter:     openShiftClusterCredentialsConverter{},
		OpenShiftClusterAdminKubeconfigConverter: openShiftClusterAdminKubeconfigConverter{},
		OpenShiftVersionConverter:                openShiftVersionConverter{},
		PlatformWorkloadIdentityRoleSetConverter: platformWorkloadIdentityRoleSetConverter{},
	}
}
