package v20260826preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/v20260826preview/generated"
)

// ExampleOpenShiftClusterCredentialsResponse returns an example
// OpenShiftClusterCredentials object that the RP might return to an end-user
func ExampleOpenShiftClusterCredentialsResponse() any {
	return &OpenShiftClusterCredentials{
		OpenShiftClusterCredentials: generated.OpenShiftClusterCredentials{
			KubeadminUsername: new("kubeadmin"),
			KubeadminPassword: new("password"),
		},
	}
}
