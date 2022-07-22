package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	ClusterDeploymentKind    = "ClusterDeployment"
	MachinePoolKind          = "MachinePool"
	SyncIdentityProviderKind = "SyncIdentityProvider"
	SyncSetKind              = "SyncSet"
)

// ClusterManagerConfiguration represents the configuration from OpenShift Cluster Manager (OCM)
type ClusterManagerConfiguration struct {
	MissingFields

	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`

	Deleting bool `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	// Supported Hive types
	ClusterDeployment    ClusterDeployment    `json:"clusterDeployment,omitempty"`
	SyncIdentityProvider SyncIdentityProvider `json:"syncIdentityProvider,omitempty"`
	SyncSet              SyncSet              `json:"syncSets,omitempty"`
	MachinePool          MachinePool          `json:"machinePool,omitempty"`

	// SystemData metadata from ARM, more info in pkg/api/openshiftcluster.go
	SystemData SystemData `json:"systemData,omitempty"`

	// Provisioning states? Determine if these will be asynchronous operations
	// or if we call it success once we write to cosmos
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`
}

// ClusterDeployment represents a Hive Cluster Deployment
type ClusterDeployment struct {
	Kind      string      `json:"kind,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// MachinePool represents a Hive Machine Pool
type MachinePool struct {
	Kind      string      `json:"kind,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// SyncSet represents a Hive Sync Set
type SyncSet struct {
	Kind      string      `json:"kind,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// SyncIdentityProvider represents a Hive Sync Identity Provider
type SyncIdentityProvider struct {
	Kind      string      `json:"kind,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}
