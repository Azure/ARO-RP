package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleOpenShiftClusterCredentialsResponse returns an example
// OpenShiftClusterCredentials object that the RP might return to an end-user
func ExampleOpenShiftClusterCredentialsResponse() *OpenShiftClusterCredentials {
	return &OpenShiftClusterCredentials{
		KubeadminUsername: "kubeadmin",
		KubeadminPassword: "password",
	}
}
