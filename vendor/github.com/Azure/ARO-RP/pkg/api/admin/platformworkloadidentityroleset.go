package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// PlatformWorkloadIdentityRoleSetList represents a List of role sets.
type PlatformWorkloadIdentityRoleSetList struct {
	// The list of role sets.
	PlatformWorkloadIdentityRoleSets []*PlatformWorkloadIdentityRoleSet `json:"value"`
}

// PlatformWorkloadIdentityRoleSet represents a mapping from the names of OCP operators to the built-in roles that should be assigned to those operator's corresponding managed identities for a particular OCP version.
type PlatformWorkloadIdentityRoleSet struct {
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
	PlatformWorkloadIdentityRoles []PlatformWorkloadIdentityRole `json:"platformWorkloadIdentityRoles,omitempty" mutable:"true"`
}

// PlatformWorkloadIdentityRole represents a mapping from a particular OCP operator to the built-in role that should be assigned to that operator's corresponding managed identity.
type PlatformWorkloadIdentityRole struct {
	// OperatorName represents the name of the operator that this role is for.
	OperatorName string `json:"operatorName,omitempty" mutable:"true" validate:"required"`

	// RoleDefinitionName represents the name of the role.
	RoleDefinitionName string `json:"roleDefinitionName,omitempty" mutable:"true" validate:"required"`

	// RoleDefinitionID represents the resource ID of the role definition.
	RoleDefinitionID string `json:"roleDefinitionId,omitempty" mutable:"true" validate:"required"`

	// ServiceAccounts represents the set of service accounts associated with the given operator, since each service account needs its own federated credential.
	ServiceAccounts []string `json:"serviceAccounts,omitempty" mutable:"true" validate:"required"`

	// SecretLocation represents the location of the in-cluster secret containing credentials for the platform workload identity.
	SecretLocation SecretLocation `json:"secretLocation,omitempty" mutable:"true" validate:"required"`
}

// SecretLocation represents the location of the in-cluster secret containing credentials for the platform workload identity.
type SecretLocation struct {
	Namespace string `json:"namespace,omitempty" mutable:"true" validate:"required"`
	Name      string `json:"name,omitempty" mutable:"true" validate:"required"`
}
