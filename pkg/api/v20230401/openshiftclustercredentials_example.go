package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleOpenShiftClusterCredentialsResponse returns an example
// OpenShiftClusterCredentials object that the RP might return to an end-user
func ExampleOpenShiftClusterCredentialsResponse() any {
	return &OpenShiftClusterCredentials{
		KubeadminUsername: "kubeadmin",
		KubeadminPassword: "password",
	}
}
