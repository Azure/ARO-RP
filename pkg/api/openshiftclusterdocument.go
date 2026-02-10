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

func (c *OpenShiftClusterDocuments) String() string {
	return encodeJSON(c)
}

func (c *OpenShiftClusterDocuments) GetCount() int {
	if c == nil {
		return 0
	}
	return c.Count
}

func (c *OpenShiftClusterDocuments) Docs() []*OpenShiftClusterDocument {
	if c == nil {
		return []*OpenShiftClusterDocument{}
	}
	return c.OpenShiftClusterDocuments
}

// OpenShiftClusterDocument represents an OpenShift cluster document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocument struct {
	MissingFields

	ID          string                 `json:"id,omitempty" deep:"-"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty" deep:"-"`
	Attachments string                 `json:"_attachments,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	Key                       string `json:"key,omitempty"`
	PartitionKey              string `json:"partitionKey,omitempty" deep:"-"`
	ClusterResourceGroupIDKey string `json:"clusterResourceGroupIdKey,omitempty"`
	ClientIDKey               string `json:"clientIdKey,omitempty"`

	Bucket int `json:"bucket,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty" deep:"-"`
	LeaseExpires int    `json:"leaseExpires,omitempty" deep:"-"`
	Dequeues     int    `json:"dequeues,omitempty"`

	AsyncOperationID string `json:"asyncOperationId,omitempty" deep:"-"`

	OpenShiftCluster *OpenShiftCluster `json:"openShiftCluster,omitempty"`

	CorrelationData *CorrelationData `json:"correlationData,omitempty" deep:"-"`
}

func (c *OpenShiftClusterDocument) String() string {
	return encodeJSON(c)
}

func (c *OpenShiftClusterDocument) GetID() string {
	return c.ID
}

func (c *OpenShiftClusterDocument) GetKey() string {
	return c.Key
}

func (c *OpenShiftClusterDocument) GetBucket() int {
	return c.Bucket
}
