package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	MachinePoolKind          = "MachinePool"
	MachinePoolType          = "MachinePools"
	SecretKind               = "Secret"
	SecretType               = "Secrets"
	SyncIdentityProviderKind = "SyncIdentityProvider"
	SyncIdentityProviderType = "SyncIdentityProviders"
	SyncSetKind              = "SyncSet"
	SyncSetType              = "SyncSets"
)

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
type Syncset struct {
}
type SyncSets struct {
}
