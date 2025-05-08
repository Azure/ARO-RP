package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftVersionList represents a list of OpenShift versions that can be
// installed.
type OpenShiftVersionList struct {
	OpenShiftVersions []*OpenShiftVersion `json:"value"`
}

type OpenShiftVersion struct {
	// The ID for the resource.
	ID string `json:"id,omitempty"`

	// Name of the resource.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// The properties for the OpenShiftVersion resource.
	Properties OpenShiftVersionProperties `json:"properties,omitempty"`
}

// OpenShiftVersionProperties represents the properties of an OpenShiftVersion.
type OpenShiftVersionProperties struct {
	// Version represents the version to create the cluster at.
	Version           string `json:"version,omitempty"`
	OpenShiftPullspec string `json:"openShiftPullspec,omitempty" mutable:"true"`
	InstallerPullspec string `json:"installerPullspec,omitempty" mutable:"true"`
	Enabled           bool   `json:"enabled" mutable:"true"`
}
