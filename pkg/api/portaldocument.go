package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// PortalDocuments represents portal documents.
// pkg/database/cosmosdb requires its definition.
type PortalDocuments struct {
	Count           int               `json:"_count,omitempty"`
	ResourceID      string            `json:"_rid,omitempty"`
	PortalDocuments []*PortalDocument `json:"Documents,omitempty"`
}

func (c *PortalDocuments) String() string {
	return encodeJSON(c)
}

// PortalDocument represents a portal document.
// pkg/database/cosmosdb requires its definition.
type PortalDocument struct {
	MissingFields

	// ID is the unique authentication token used by the SRE
	ID          string                 `json:"id,omitempty"`
	ResourceID  string                 `json:"_rid,omitempty"`
	Timestamp   int                    `json:"_ts,omitempty"`
	Self        string                 `json:"_self,omitempty"`
	ETag        string                 `json:"_etag,omitempty"`
	Attachments string                 `json:"_attachments,omitempty"`
	TTL         int                    `json:"ttl,omitempty"`
	LSN         int                    `json:"_lsn,omitempty"`
	Metadata    map[string]interface{} `json:"_metadata,omitempty"`

	Portal *Portal `json:"portal,omitempty"`
}

func (c *PortalDocument) String() string {
	return encodeJSON(c)
}
