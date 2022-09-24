package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// SyncSetList represents a list of SyncSets for a given cluster.
type SyncSetList struct {
	SyncSets []*SyncSet `json:"value"`
}

type ClusterManagerConfigurationList struct {
	ClusterManagerConfigurations []*ClusterManagerConfiguration `json:"value"`
}

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {
	// ID is the unique identifier for the cluster manager configuration
	ID                string                                `json:"id,omitempty"`
	Name              string                                `json:"name,omitempty"`
	ClusterResourceID string                                `json:"clusterResourceId,omitempty"`
	Properties        ClusterManagerConfigurationProperties `json:"properties,omitempty"`
	SystemData        *SystemData                           `json:"systemData,omitempty"`
}

// ClusterManagerConfigurationProperties houses the payloads the frontend receives for all OCM resources
// we store them as a byte slice in cosmos
type ClusterManagerConfigurationProperties struct {
	Resources []byte `json:"resources,omitempty"`
}

// SyncSet represents a SyncSet for an Azure Red Hat OpenShift Cluster.
type SyncSet struct {
	// Required resource properties in ARM
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The SyncSets properties
	Properties SyncSetProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncSetProperties struct {
	// Resources represents the SyncSets configuration.
	Resources string `json:"resources,omitempty"`
}

// MachinePoolList represents a list of MachinePools
type MachinePoolList struct {
	// The list of MachinePools.
	MachinePools []*MachinePool `json:"value"`
}

// MachinePool represents a MachinePool
type MachinePool struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The MachinePool Properties
	Properties MachinePoolProperties `json:"properties,omitempty"`
}

// MachinePoolProperties represents the properties of a MachinePool
type MachinePoolProperties struct {
	Resources string `json:"resources,omitempty"`
}

// SyncIdentityProviderList represents a list of SyncIdentityProvider
type SyncIdentityProviderList struct {
	// The list of SyncIdentityProvider.
	SyncIdentityProviders []*SyncIdentityProvider `json:"value"`
}

// SyncIdentityProvider represents a SyncIdentityProvider
type SyncIdentityProvider struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The SyncIdentityProvider Properties
	Properties SyncIdentityProviderProperties `json:"properties,omitempty"`
}

// SyncIdentityProviderProperties represents the properties of a SyncIdentityProvider
type SyncIdentityProviderProperties struct {
	// The SyncIdentityProvider Resources.
	Resources string `json:"resources,omitempty"`
}

// SecretList represents a list of Secrets
type SecretList struct {
	// The list of Secrets.
	Secrets []*Secret `json:"value"`
}

// Secret represents a secret.
type Secret struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The resource type.
	Type string `json:"type,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	// The Secret Properties
	Properties SecretProperties `json:"properties,omitempty"`
}

// SecretProperties represents the properties of a Secret
type SecretProperties struct {
	// The Secrets Resources.
	SecretResources SecureString `json:"secretResources,omitempty"`
}
