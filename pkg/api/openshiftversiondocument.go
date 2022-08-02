package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftVersionDocuments represents OpenShift version specification documents.
// pkg/database/cosmosdb requires its definition.
type OpenShiftVersionDocuments struct {
	Count                     int                         `json:"_count,omitempty"`
	ResourceID                string                      `json:"_rid,omitempty"`
	OpenShiftVersionDocuments []*OpenShiftVersionDocument `json:"Documents,omitempty"`
}

func (c *OpenShiftVersionDocuments) String() string {
	return encodeJSON(c)
}

// OpenShiftVersionDocument represents an OpenShift version specification document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftVersionDocument struct {
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

	OpenShiftVersion *OpenShiftVersion `json:"openShiftVersion,omitempty"`
}

func (c *OpenShiftVersionDocument) String() string {
	return encodeJSON(c)
}
