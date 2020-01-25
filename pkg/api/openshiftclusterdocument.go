package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterDocuments represents OpenShift cluster documents.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocuments struct {
	Count                     int                         `json:"_count,omitempty"`
	ResourceID                string                      `json:"_rid,omitempty"`
	OpenShiftClusterDocuments []*OpenShiftClusterDocument `json:"Documents,omitempty"`
}

// OpenShiftClusterDocument represents an OpenShift cluster document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocument struct {
	MissingFields

	ID          string                 `json:"id,omitempty"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty"`
	Attachments string                 `json:"_attachments,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	Key                       string `json:"key,omitempty"`
	PartitionKey              string `json:"partitionKey,omitempty"`
	ClusterResourceGroupIDKey string `json:"clusterResourceGroupIdKey,omitempty"`
	ClientIDKey               string `json:"clientIdKey,omitempty"`

	Bucket int `json:"bucket,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty"`
	LeaseExpires int    `json:"leaseExpires,omitempty"`
	Dequeues     int    `json:"dequeues,omitempty"`

	AsyncOperationID string `json:"asyncOperationId,omitempty"`

	OpenShiftCluster *OpenShiftCluster `json:"openShiftCluster,omitempty"`
}
