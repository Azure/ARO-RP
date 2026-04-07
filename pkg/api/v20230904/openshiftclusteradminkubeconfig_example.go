package v20230904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleOpenShiftClusterAdminKubeconfigResponse returns an example
// OpenShiftClusterAdminKubeconfig object that the RP might return to an end-user
func ExampleOpenShiftClusterAdminKubeconfigResponse() any {
	return &OpenShiftClusterAdminKubeconfig{
		Kubeconfig: []byte("{}"),
	}
}
