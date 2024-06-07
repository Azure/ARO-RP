package v20240812preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// PlatformWorkloadIdentityRoleSetList represents a List of role sets.
type PlatformWorkloadIdentityRoleSetList struct {
	// The list of role sets.
	PlatformWorkloadIdentityRoleSets []*PlatformWorkloadIdentityRoleSet `json:"value"`

	// Next Link to next operation.
	NextLink string `json:"nextLink,omitempty"`
}

// PlatformWorkloadIdentityRoleSet represents a mapping from the names of OCP operators to the built-in roles that should be assigned to those operator's corresponding managed identities for a particular OCP version.
type PlatformWorkloadIdentityRoleSet struct {
	proxyResource bool

	// The ID for the resource.
	ID string `json:"id,omitempty" mutable:"case"`

	// Name of the resource.
	Name string `json:"name,omitempty" mutable:"case"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// The properties for the PlatformWorkloadIdentityRoleSet resource.
	Properties PlatformWorkloadIdentityRoleSetProperties `json:"properties,omitempty"`
}

// PlatformWorkloadIdentityRoleSetProperties represents the properties of a PlatformWorkloadIdentityRoleSet resource.
type PlatformWorkloadIdentityRoleSetProperties struct {
	// OpenShiftVersion represents the version associated with this set of roles.
	OpenShiftVersion string `json:"openShiftVersion,omitempty"`

	// PlatformWorkloadIdentityRoles represents the set of roles associated with this version.
	PlatformWorkloadIdentityRoles []PlatformWorkloadIdentityRole `json:"platformWorkloadIdentityRoles,omitempty"`
}

// PlatformWorkloadIdentityRole represents a mapping from a particular OCP operator to the built-in role that should be assigned to that operator's corresponding managed identity.
type PlatformWorkloadIdentityRole struct {
	// OperatorName represents the name of the operator that this role is for.
	OperatorName string `json:"operatorName,omitempty"`

	// RoleDefinitionName represents the name of the role.
	RoleDefinitionName string `json:"roleDefinitionName,omitempty"`

	// RoleDefinitionID represents the resource ID of the role definition.
	RoleDefinitionID string `json:"roleDefinitionId,omitempty"`
}
