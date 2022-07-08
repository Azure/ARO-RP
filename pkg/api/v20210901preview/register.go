package v20210901preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

// APIVersion contains a version string as it will be used by clients
const APIVersion = "2021-09-01-preview"

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

func init() {
	api.APIs[APIVersion] = &api.Version{
		OpenShiftClusterConverter: func() api.OpenShiftClusterConverter {
			return &openShiftClusterConverter{}
		},
		OpenShiftClusterDocumentConverter: func() api.OpenShiftClusterDocumentConverter {
			return &openShiftClusterDocumentConverter{}
		},
		OpenShiftClusterStaticValidator: func(location, domain string, requireD2sV3Workers bool, resourceID string) api.OpenShiftClusterStaticValidator {
			return &openShiftClusterStaticValidator{
				location:            location,
				domain:              domain,
				requireD2sV3Workers: requireD2sV3Workers,
				resourceID:          resourceID,
			}
		},
		OpenShiftClusterCredentialsConverter: func() api.OpenShiftClusterCredentialsConverter {
			return &openShiftClusterCredentialsConverter{}
		},
		OpenShiftClusterAdminKubeconfigConverter: func() api.OpenShiftClusterAdminKubeconfigConverter {
			return &openShiftClusterAdminKubeconfigConverter{}
		},
	}
}
