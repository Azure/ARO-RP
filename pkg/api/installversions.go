package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// InstallVersionList represents a List of OpenShift installable versions.
type InstallVersionList struct {
	// List of InstallVersion
	InstallVersions *[]InstallVersion
}

// InstallVersion is going to be a proxy resource in our versioned API's
// This requires us to model the structs in the following manner:
type InstallVersion struct {
	Name       string                   `json:"name,omitempty"`
	ID         string                   `json:"id,omitempty"`
	Type       string                   `json:"type,omitempty"`
	Properties InstallVersionProperties `json:"properties,omitempty"`
}

type InstallVersionProperties struct {
	Version string `json:"version,omitempty"`
}
