package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// MonitorDocuments represents monitor documents.
// pkg/database/cosmosdb requires its definition.
type MonitorDocuments struct {
	Count            int                `json:"_count,omitempty"`
	ResourceID       string             `json:"_rid,omitempty"`
	MonitorDocuments []*MonitorDocument `json:"Documents,omitempty"`
}

// MonitorDocument represents a monitor document.
// pkg/database/cosmosdb requires its definition.
type MonitorDocument struct {
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

	Monitor *Monitor `json:"monitor,omitempty"`
}

func (c *MonitorDocument) GetID() string {
	return c.ID
}
