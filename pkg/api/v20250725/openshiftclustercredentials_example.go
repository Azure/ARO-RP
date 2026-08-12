package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

// ExampleOpenShiftClusterCredentialsResponse returns an example
// OpenShiftClusterCredentials object that the RP might return to an end-user
func ExampleOpenShiftClusterCredentialsResponse() interface{} {
	return &OpenShiftClusterCredentials{
		OpenShiftClusterCredentials: generated.OpenShiftClusterCredentials{
			KubeadminUsername: pointerutils.ToPtr("kubeadmin"),
			KubeadminPassword: pointerutils.ToPtr("password"),
		},
	}
}
