package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterCredentials represents an OpenShift cluster's credentials.
type OpenShiftClusterCredentials struct {
	// The username for the kubeadmin user.
	KubeadminUsername string `json:"kubeadminUsername,omitempty"`

	// The password for the kubeadmin user.
	KubeadminPassword string `json:"kubeadminPassword,omitempty"`
}
