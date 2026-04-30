package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// PlatformWorkloadIdentityRoleSet represents a mapping from the names of OCP operators to the built-in roles that should be assigned to those operator's corresponding managed identities for a particular OCP version.
type PlatformWorkloadIdentityRoleSet struct {
	MissingFields

	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Deleting bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	Properties PlatformWorkloadIdentityRoleSetProperties `json:"properties,omitempty"`
}

// PlatformWorkloadIdentityRoleSetProperties represents the properties of a PlatformWorkloadIdentityRoleSet resource.
type PlatformWorkloadIdentityRoleSetProperties struct {
	OpenShiftVersion              string                         `json:"openShiftVersion,omitempty"`
	PlatformWorkloadIdentityRoles []PlatformWorkloadIdentityRole `json:"platformWorkloadIdentityRoles,omitempty"`
}

// PlatformWorkloadIdentityRole represents a mapping from a particular OCP operator to the built-in role that should be assigned to that operator's corresponding managed identity.
type PlatformWorkloadIdentityRole struct {
	OperatorName       string         `json:"operatorName,omitempty"`
	RoleDefinitionName string         `json:"roleDefinitionName,omitempty"`
	RoleDefinitionID   string         `json:"roleDefinitionId,omitempty"`
	ServiceAccounts    []string       `json:"serviceAccounts,omitempty"`
	SecretLocation     SecretLocation `json:"secretLocation,omitempty"`
}

// SecretLocation represents the location of the in-cluster secret containing credentials for the platform workload identity.
type SecretLocation struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

const PlatformWorkloadIdentityRoleSetsType = "Microsoft.RedHatOpenShift/PlatformWorkloadIdentityRoleSet"
