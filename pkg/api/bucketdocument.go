package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// BucketServiceDocuments represents Bucket documents.
// pkg/database/cosmosdb requires its definition.
type BucketServiceDocuments struct {
	Count                  int                      `json:"_count,omitempty"`
	ResourceID             string                   `json:"_rid,omitempty"`
	BucketServiceDocuments []*BucketServiceDocument `json:"Documents,omitempty"`
}

// BucketServiceDocument represents a BucketService document.
// pkg/database/cosmosdb requires its definition.
type BucketServiceDocument struct {
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

	LeaseOwner   string `json:"leaseOwner,omitempty"`
	LeaseExpires int    `json:"leaseExpires,omitempty"`

	ServiceName string   `json:"serviceName,omitempty"`
	ServiceRole string   `json:"serviceRole,omitempty"`
	Buckets     []string `json:"buckets,omitempty"`
}
