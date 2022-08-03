package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	MachinePoolKind          = "MachinePool"
	MachinePoolType          = "MachinePools"
	SyncIdentityProviderKind = "SyncIdentityProvider"
	SyncIdentityProviderType = "SyncIdentityProviders"
	SyncSetKind              = "SyncSet"
	SyncSetType              = "SyncSets"
)

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {
	MissingFields

	// ID is the unique identifier for the cluster manager configuration
	ID string `json:"id,omitempty"`

	ClusterResourceId string `json:"clusterResourceId,omitempty"`
	Kind              string `json:"kind,omitempty"`
	Resources         []byte `json:"resources,omitempty"`
	Deleting          bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	// SystemData metadata from ARM, more info in pkg/api/openshiftcluster.go
	SystemData *SystemData `json:"systemData,omitempty"`
}
