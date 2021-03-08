package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// GatewayDocuments represents gateway documents.
// pkg/database/cosmosdb requires its definition.
type GatewayDocuments struct {
	Count            int                `json:"_count,omitempty"`
	ResourceID       string             `json:"_rid,omitempty"`
	GatewayDocuments []*GatewayDocument `json:"Documents,omitempty"`
}

func (c *GatewayDocuments) String() string {
	return encodeJSON(c)
}

// GatewayDocument represents a gateway document.
// pkg/database/cosmosdb requires its definition.
type GatewayDocument struct {
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

	Gateway *Gateway `json:"gateway,omitempty"`
}

func (c *GatewayDocument) String() string {
	return encodeJSON(c)
}
