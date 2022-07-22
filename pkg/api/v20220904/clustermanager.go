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

// SyncSetList represents a list of SyncSets
type SyncSetList struct {
	SyncSets []*SyncSet `json:"value"`
	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

type ClusterManagerConfigurationList struct {
	ClusterManagerConfigurations []*ClusterManagerConfiguration `json:"value"`

	NextLink string `json:"nextLink,omitempty"`
}

type ClusterManagerConfiguration struct {
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
	Resources interface{} `json:"resources,omitempty"`
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

	// // SystemData metadata relating to this resource.
	// SystemData *SystemData `json:"systemData,omitempty"`

	// The Syncsets properties
	Properties SyncSetProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncSetProperties struct {
	// The parent Azure Red Hat OpenShift resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// APIVersion for the SyncSet.
	APIVersion string `json:"apiVersion,omitempty"`

	// SyncSet kind.
	Kind string `json:"kind,omitempty"`

	// Metadata for the SyncSet.
	Metadata map[string]string `json:"metadata,omitempty"`

	// The SyncSet Specification.
	Spec SyncSetSpec `json:"spec,omitempty"`

	// ClusterDeploymentRefs map SyncSets to a Hive Cluster Deployment.
	ClusterDeploymentRefs []string `json:"clusterDeploymentRefs,omitempty"`

	// Resources represents the SyncSets configuration.
	Resources map[string]string `json:"resources,omitempty"`

	// The status of the object.
	Status string `json:"status,omitempty"`

	// Resources []byte `json:"resources,omitempty"`
}

type SyncSetSpec struct {
	// ClusterDeploymentRefs map SyncSets to a Hive Cluster Deployment.
	ClusterDeploymentRefs []string `json:"clusterDeploymentRefs,omitempty"`

	// Resources represents the SyncSets configuration.
	Resources map[string]interface{} `json:"resources,omitempty"`

	// The status of the object.
	Status string `json:"status,omitempty"`
}

// MachinePool represents a MachinePool
type MachinePool struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The parent cluster resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	Properties MachinePoolProperties `json:"properties,omitempty"`
}

// MachinePoolProperties represents the properties of a MachinePool
type MachinePoolProperties struct {
	Resources interface{} `json:"resources,omitempty"`
}

// SyncIdentityProvider represents a SyncIdentityProvider
type SyncIdentityProvider struct {
	// The Resource ID.
	ID string `json:"id,omitempty"`

	// The resource name.
	Name string `json:"name,omitempty"`

	// The parent cluster resourceID.
	ClusterResourceId string `json:"clusterResourceId,omitempty"`

	// SystemData metadata relating to this resource.
	SystemData *SystemData `json:"systemData,omitempty"`

	Properties SyncIdentityProviderProperties `json:"properties,omitempty"`
}

// SyncSetProperties represents the properties of a SyncSet
type SyncIdentityProviderProperties struct {
	Resources interface{} `json:"resources,omitempty"`
}
