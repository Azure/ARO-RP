package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

// OpenShiftVersionList represents a List of available versions.
type OpenShiftVersionList struct {
	// The List of available versions.
	OpenShiftVersions []*OpenShiftVersion `json:"value"`

	// Next Link to next operation.
	NextLink string `json:"nextLink,omitempty"`
}

// OpenShiftVersion represents an OpenShift version that can be installed.
type OpenShiftVersion struct {
	generated.OpenShiftVersion
}
