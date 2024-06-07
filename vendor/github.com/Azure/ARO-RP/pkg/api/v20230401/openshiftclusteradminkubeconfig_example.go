package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ExampleOpenShiftClusterAdminKubeconfigResponse returns an example
// OpenShiftClusterAdminKubeconfig object that the RP might return to an end-user
func ExampleOpenShiftClusterAdminKubeconfigResponse() interface{} {
	return &OpenShiftClusterAdminKubeconfig{
		Kubeconfig: []byte("{}"),
	}
}
