package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// AsyncOperationDocuments represents asyncOperation documents.
// pkg/database/cosmosdb requires its definition.
type AsyncOperationDocuments struct {
	Count                   int                       `json:"_count,omitempty"`
	ResourceID              string                    `json:"_rid,omitempty"`
	AsyncOperationDocuments []*AsyncOperationDocument `json:"Documents,omitempty"`
}

func (c *AsyncOperationDocuments) String() string {
	return encodeJSON(c)
}

// AsyncOperationDocument represents a asyncOperation document.
// pkg/database/cosmosdb requires its definition.
type AsyncOperationDocument struct {
	MissingFields

	ID          string                 `json:"id,omitempty" deep:"-"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty" deep:"-"`
	Attachments string                 `json:"_attachments,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	AsyncOperation *AsyncOperation `json:"asyncOperation,omitempty"`

	OpenShiftClusterKey string            `json:"openShiftClusterKey,omitempty"`
	OpenShiftCluster    *OpenShiftCluster `json:"openShiftCluster,omitempty"`
}

func (c *AsyncOperationDocument) String() string {
	return encodeJSON(c)
}
