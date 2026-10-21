package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
)

// PlatformWorkloadIdentityRoleSetList represents a List of role sets.
type PlatformWorkloadIdentityRoleSetList struct {
	// The list of role sets.
	PlatformWorkloadIdentityRoleSets []*PlatformWorkloadIdentityRoleSet `json:"value"`

	// Next Link to next operation.
	NextLink string `json:"nextLink,omitempty"`
}

// PlatformWorkloadIdentityRoleSet represents a mapping from the names of OCP operators to the built-in roles that should be assigned to those operator's corresponding managed identities for a particular OCP version.
type PlatformWorkloadIdentityRoleSet struct {
	generated.PlatformWorkloadIdentityRoleSet
}
