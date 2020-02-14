package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ResourceDocuments represents external resource document.
// pkg/resources requires its definition.
type ResourceDocuments struct {
	Count             int                 `json:"_count,omitempty"`
	ResourceID        string              `json:"_rid,omitempty"`
	ResourceDocuments []*ResourceDocument `json:"Documents,omitempty"`
}

// ResourceDocument represents a resource document
// pkg/resources requires its definition.
type ResourceDocument struct {
	MissingFields

	ID          string                 `json:"id,omitempty"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty"`
	Attachments string                 `json:"_attachments,omitempty"`
	TTL         int                    `json:"ttl,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty"`
	LeaseExpires int    `json:"leaseExpires,omitempty"`

	Resource *Resource `json:"resource,omitempty"`
}
