package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterCredentials represents an OpenShift cluster's credentials
type OpenShiftClusterCredentials struct {
	// The password for the kubeadmin user
	KubeadminPassword string `json:"kubeadminPassword,omitempty"`
}
