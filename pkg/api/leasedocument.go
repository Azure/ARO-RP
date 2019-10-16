package api

import (
	uuid "github.com/satori/go.uuid"
)

// LeaseDocuments represents a lease documents.
// pkg/database/cosmosdb requires its definition.
type LeaseDocuments struct {
	Count          int              `json:"_count,omitempty"`
	ResourceID     string           `json:"_rid,omitempty"`
	LeaseDocuments []*LeaseDocument `json:"Documents,omitempty"`
}

// LeaseDocument represents a lease document.
// pkg/database/cosmosdb requires its definition.
type LeaseDocument struct {
	MissingFields

	ID          string `json:"id,omitempty"`
	ResourceID  string `json:"_rid,omitempty"`
	Timestamp   int    `json:"_ts,omitempty"`
	Self        string `json:"_self,omitempty"`
	ETag        string `json:"_etag,omitempty"`
	Attachments string `json:"_attachments,omitempty"`

	TTL int `json:"ttl,omitempty"`

	Holder uuid.UUID `json:"holder,omitempty"`
}
