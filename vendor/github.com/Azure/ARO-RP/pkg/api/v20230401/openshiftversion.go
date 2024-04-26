package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftVersionList represents a List of available versions.
type OpenShiftVersionList struct {
	// The List of available versions.
	OpenShiftVersions []*OpenShiftVersion `json:"value"`

	// Next Link to next operation.
	NextLink string `json:"nextLink,omitempty"`
}

// OpenShiftVersion represents an OpenShift version that can be installed.
type OpenShiftVersion struct {
	proxyResource bool

	// The ID for the resource.
	ID string `json:"id,omitempty" mutable:"case"`

	// Name of the resource.
	Name string `json:"name,omitempty" mutable:"case"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// The properties for the OpenShiftVersion resource.
	Properties OpenShiftVersionProperties `json:"properties,omitempty"`
}

// OpenShiftVersionProperties represents the properties of an OpenShiftVersion.
type OpenShiftVersionProperties struct {
	// Version represents the version to create the cluster at.
	Version string `json:"version,omitempty"`
}
