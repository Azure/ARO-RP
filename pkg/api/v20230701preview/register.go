package v20230701preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// APIVersion contains a version string as it will be used by clients
const APIVersion = "2023-07-01-preview"

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
		SyncSetConverter:                         syncSetConverter{},
		MachinePoolConverter:                     machinePoolConverter{},
		SyncIdentityProviderConverter:            syncIdentityProviderConverter{},
		SecretConverter:                          secretConverter{},
		ClusterManagerStaticValidator:            clusterManagerStaticValidator{},
	}
}
