package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

func init() {
	api.APIs["2019-12-31-preview"] = &api.Version{
		OpenShiftClusterConverter: func() api.OpenShiftClusterConverter {
			return &openShiftClusterConverter{}
		},
		OpenShiftClusterStaticValidator: func(location, resourceID string) api.OpenShiftClusterStaticValidator {
			return &openShiftClusterStaticValidator{
				location:   location,
				resourceID: resourceID,
			}
		},
		OpenShiftClusterCredentialsConverter: func() api.OpenShiftClusterCredentialsConverter {
			return &openShiftClusterCredentialsConverter{}
		},
	}
}
