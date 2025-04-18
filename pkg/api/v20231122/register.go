package v20231122

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// APIVersion contains a version string as it will be used by clients
const APIVersion = "2023-11-22"

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

func init() {
	api.APIs[APIVersion] = &api.Version{
		OpenShiftClusterConverter:                openShiftClusterConverter{},
		OpenShiftClusterStaticValidator:          openShiftClusterStaticValidator{},
		OpenShiftClusterCredentialsConverter:     openShiftClusterCredentialsConverter{},
		OpenShiftClusterAdminKubeconfigConverter: openShiftClusterAdminKubeconfigConverter{},
		OpenShiftVersionConverter:                openShiftVersionConverter{},
		OperationList: api.OperationList{
			Operations: []api.Operation{
				api.OperationResultsRead,
				api.OperationStatusRead,
				api.OperationRead,
				api.OperationOpenShiftClusterRead,
				api.OperationOpenShiftClusterWrite,
				api.OperationOpenShiftClusterDelete,
				api.OperationOpenShiftClusterListCredentials,
				api.OperationOpenShiftClusterListAdminCredentials,
				api.OperationListInstallVersions,
				api.OperationOpenShiftClusterGetDetectors,
			},
		},
	}
}
