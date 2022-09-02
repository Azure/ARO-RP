package installversion

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftCluster represents the portion of the OpenShift cluster
// representation which provides the cluster installation version.
type openShiftCluster struct {
	Properties openShiftClusterProperties `json:"properties,omitempty"`
}

// OpenShiftClusterProperties represents an OpenShift cluster's properties.
type openShiftClusterProperties struct {
	// The cluster install version.
	InstallVersion string `json:"installVersion,omitempty"`
}
