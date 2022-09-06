package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// SyncSetList represents a list of SyncSets for a given cluster.
type SyncSetList struct {
	Syncsets []*SyncSet `json:"value"`
	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

type ClusterManagerConfigurationList struct {
	ClusterManagerConfigurations []*ClusterManagerConfiguration `json:"value"`

	NextLink string `json:"nextLink,omitempty"`
}

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {
	MissingFields

	// ID is the unique identifier for the cluster manager configuration
	ID                string                                `json:"id,omitempty"`
	Name              string                                `json:"name,omitempty"`
	ClusterResourceId string                                `json:"clusterResourceId,omitempty"`
	Deleting          bool                                  `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
	Properties        ClusterManagerConfigurationProperties `json:"properties,omitempty"`
	// SystemData metadata from ARM, more info in pkg/api/openshiftcluster.go
	SystemData *SystemData `json:"systemData,omitempty"`
}

type ClusterManagerConfigurationProperties struct {
	Resources []byte `json:"resources,omitempty"`
}

// SyncSet represents a SyncSet for an Azure Red Hat OpenShift Cluster.
type SyncSet struct {
	MissingFields

	// ID, Name and Type are cased as the user provided them at create time.
	// ID, Name, Type and Location are immutable.
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`

	// The Syncsets properties
	Properties SyncSetProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncSetProperties struct {
	// The parent Azure Red Hat OpenShift resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// Resources represents the SyncSets configuration.
	Resources string `json:"resources,omitempty"`
}

// MachinePool represents a MachinePool
type MachinePool struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The MachinePool properties.
	Properties MachinePoolProperties `json:"properties,omitempty"`
}

// MachinePoolProperties represents the properties of a MachinePool
type MachinePoolProperties struct {

	// The parent cluster resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// Resources represents the machinepools configuration.
	Resources []byte `json:"resources,omitempty"`
}

// SyncIdentityProvider represents a SyncIdentityProvider
type SyncIdentityProvider struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The parent cluster resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// The SyncIdentityProvider properties.
	Properties SyncIdentityProviderProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncIdentityProviderProperties struct {
	MissingFields
	Resources []byte `json:"resources,omitempty"`
}

// // HiveSecret represents a hive secret.
// type HiveSecret struct {

// }

// // SyncSetProperties represents the properties of a SyncSet
// type HiveSecretProperties struct {
// 	MissingFields
// 	Resources []byte `json:"resources,omitempty"`
// }
