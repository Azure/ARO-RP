package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
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
		OpenShiftClusterValidator: func(env env.Interface, resourceID string) api.OpenShiftClusterValidator {
			return &openShiftClusterValidator{
				sv: openShiftClusterStaticValidator{
					location:   env.Location(),
					resourceID: resourceID,
				},
				dv: openShiftClusterDynamicValidator{
					env: env,
				},
			}
		},
		OpenShiftClusterCredentialsConverter: func() api.OpenShiftClusterCredentialsConverter {
			return &openShiftClusterCredentialsConverter{}
		},
	}
}
