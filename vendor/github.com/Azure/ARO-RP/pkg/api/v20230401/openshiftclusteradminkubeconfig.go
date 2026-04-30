package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterAdminKubeconfig represents an OpenShift cluster's admin kubeconfig.
type OpenShiftClusterAdminKubeconfig struct {
	// The base64-encoded kubeconfig file.
	Kubeconfig []byte `json:"kubeconfig,omitempty"`
}
