package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftVersionList represents a list of OpenShift versions that can be
// installed.
type OpenShiftVersionList struct {
	OpenShiftVersions []*OpenShiftVersion `json:"value"`
}

type OpenShiftVersion struct {
	Version           string `json:"version,omitempty"`
	OpenShiftPullspec string `json:"openShiftPullspec,omitempty" mutable:"true"`
	InstallerPullspec string `json:"installerPullspec,omitempty" mutable:"true"`
	Enabled           bool   `json:"enabled,omitempty" mutable:"true"`
}
