package v20220904

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OCM Kinds supported
const (
	MachinePoolKind          = "MachinePool"
	SyncIdentityProviderKind = "SyncIdentityProvider"
	SyncSetKind              = "SyncSet"
	SecretKind               = "Secret"
)

type ClusterManagerConfigurationsList struct {
	ClusterManagerConfigurations []*ClusterManagerConfiguration `json:"value"`

	NextLink string `json:"nextLink,omitempty"`
}

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {

	// ID is the unique identifier for the cluster manager configuration
	ID                string                                `json:"id,omitempty"`
	ClusterResourceId string                                `json:"clusterResourceId,omitempty"`
	Name              string                                `json:"name,omitempty"`
	Properties        ClusterManagerConfigurationProperties `json:"properties,omitempty"`
	Deleting          bool                                  `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
}

// ClusterManagerConfigurationProperties represents properties of a cluster manager configuration
type ClusterManagerConfigurationProperties struct {
	Resources interface{} `json:"resources,omitempty"`
}
type Syncset struct{}
type SyncSets struct{}
