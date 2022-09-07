package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ClusterManagerConfigurationDocument represents OpenShift cluster manager configuration documents.
// pkg/database/cosmosdb requires its definition.
type ClusterManagerConfigurationDocuments struct {
	Count                                int                                    `json:"_count,omitempty"`
	ResourceID                           string                                 `json:"_rid,omitempty"`
	ClusterManagerConfigurationDocuments []*ClusterManagerConfigurationDocument `json:"Documents,omitempty"`
}

// String returns a JSON representation of the OpenShiftClusterManagerConfigurationDocuments struct.
func (c *ClusterManagerConfigurationDocuments) String() string {
	return encodeJSON(c)
}

// OpenShiftClusterManagerConfigurationDocument represents an OpenShift cluster manager configuration document.
// pkg/database/cosmosdb requires its definition.
type ClusterManagerConfigurationDocument struct {
	MissingFields

	ID          string                 `json:"id,omitempty"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty" deep:"-"`
	Attachments string                 `json:"_attachments,omitempty"`
	TTL         int                    `json:"ttl,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	Key          string `json:"key,omitempty"`
	PartitionKey string `json:"partitionKey,omitempty" deep:"-"`
	Deleting     bool   `json:"deleting,omitempty"` // https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes

	ClusterManagerConfiguration *ClusterManagerConfiguration `json:"clusterManagerConfiguration,omitempty"`

	SyncIdentityProvider *SyncIdentityProvider `json:"syncIdentityProvider,omitempty"`
	SyncSet              *SyncSet              `json:"syncSet,omitempty"`
	MachinePool          *MachinePool          `json:"machinePool,omitempty"`
	Secret               *Secret               `json:"secret,omitempty"`

	CorrelationData *CorrelationData `json:"correlationData,omitempty" deep:"-"`
}

// String returns a JSON representation of the OpenShiftClusterManagerConfigurationDocument struct.
func (c *ClusterManagerConfigurationDocument) String() string {
	return encodeJSON(c)
}
