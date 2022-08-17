package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftVersion represents an OpenShift version that can be installed
type OpenShiftVersion struct {
	MissingFields

	ID       string `json:"id,omitempty"`
	Deleting bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	Version           string `json:"version,omitempty"`
	OpenShiftPullspec string `json:"openShiftPullspec,omitempty"`
	InstallerPullspec string `json:"installerPullspec,omitempty"`
	Enabled           bool   `json:"enabled,omitempty"`
}
