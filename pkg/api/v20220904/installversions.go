package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// InstallVersionList represents a List of OpenShift installable versions.
type InstallVersionList struct {
	// List of InstallVersion
	InstallVersions []*InstallVersion `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// InstallVersion is going to be a proxy resource in our versioned API's
type InstallVersion struct {
	// this is an "unused" bool the type walker keys off during swagger generation
	// TODO refactor this logic to be contained within the swagger package not api packages
	proxyResource bool

	// The resource name.
	Name string `json:"name,omitempty"`

	// The ID is the unique identifier for the installversion
	ID string `json:"id,omitempty"`

	// The InstallVersion properties
	Properties InstallVersionProperties `json:"properties,omitempty"`
}

type InstallVersionProperties struct {
	// Version is the available OpenShift version to install.
	Version string `json:"version,omitempty"`
}
