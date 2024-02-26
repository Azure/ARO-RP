package v20230401

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// SyncSetList represents a list of SyncSets
type SyncSetList struct {
	// The list of syncsets.
	SyncSets []*SyncSet `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// SyncSet represents a SyncSet for an Azure Red Hat OpenShift Cluster.
type SyncSet struct {
	// This is a flag used during the swagger generation typewalker to
	// signal that it should be marked as a proxy resource and
	// not a tracked ARM resource.
	proxyResource bool

	// The resource ID.
	ID string `json:"id,omitempty" mutable:"case"`

	// The resource name.
	Name string `json:"name,omitempty" mutable:"case"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The Syncsets properties
	Properties SyncSetProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncSetProperties struct {
	// Resources represents the SyncSets configuration.
	Resources string `json:"resources,omitempty"`
}

// MachinePoolList represents a list of MachinePools
type MachinePoolList struct {
	// The list of Machine Pools.
	MachinePools []*MachinePool `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// MachinePool represents a MachinePool
type MachinePool struct {
	// This is a flag used during the swagger generation typewalker to
	// signal that it should be marked as a proxy resource and
	// not a tracked ARM resource.
	proxyResource bool

	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The MachinePool Properties
	Properties MachinePoolProperties `json:"properties,omitempty"`
}

// MachinePoolProperties represents the properties of a MachinePool
type MachinePoolProperties struct {
	Resources string `json:"resources,omitempty"`
}

// SyncSetList represents a list of SyncSets
type SyncIdentityProviderList struct {
	// The list of sync identity providers
	SyncIdentityProviders []*SyncIdentityProvider `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// SyncIdentityProvider represents a SyncIdentityProvider
type SyncIdentityProvider struct {
	// This is a flag used during the swagger generation typewalker to
	// signal that it should be marked as a proxy resource and
	// not a tracked ARM resource.
	proxyResource bool

	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The SyncIdentityProvider Properties
	Properties SyncIdentityProviderProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncIdentityProviderProperties struct {
	Resources string `json:"resources,omitempty"`
}

// SecretList represents a list of Secrets
type SecretList struct {
	// The list of secrets.
	Secrets []*Secret `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

// Secret represents a secret.
type Secret struct {
	// This is a flag used during the swagger generation typewalker to
	// signal that it should be marked as a proxy resource and
	// not a tracked ARM resource.
	proxyResource bool

	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty" mutable:"case"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The Secret Properties
	Properties SecretProperties `json:"properties,omitempty"`
}

// SecretProperties represents the properties of a Secret
type SecretProperties struct {
	// The Secrets Resources.
	SecretResources string `json:"secretResources,omitempty"`
}
