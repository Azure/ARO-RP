package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// see openshift/installer/pkg/asset/installconfig

// Networking defines the pod network provider in the cluster.
type Networking struct {
	// NetworkType is the type of network to install. The default is OpenShiftSDN
	NetworkType string `json:"networkType,omitempty"`
}

// InstallConfig is the configuration for an OpenShift install.
type Config struct {

	// Networking is the configuration for the pod network provider in
	// the cluster.
	*Networking `json:"networking,omitempty"`
}

// InstallConfig generates the install-config.yaml file.
type InstallConfig struct {
	Config *Config `json:"config"`
}
