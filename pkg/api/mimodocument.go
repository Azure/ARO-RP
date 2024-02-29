package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type MaintenanceManifestDocuments struct {
	Count                        int                            `json:"_count,omitempty"`
	ResourceID                   string                         `json:"_rid,omitempty"`
	MaintenanceManifestDocuments []*MaintenanceManifestDocument `json:"Documents,omitempty"`
}

func (e *MaintenanceManifestDocuments) String() string {
	return encodeJSON(e)
}

type MaintenanceManifestDocument struct {
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

	ClusterID           string               `json:"clusterID,omitempty"`
	MaintenanceManifest *MaintenanceManifest `json:"maintenanceManifest,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty" deep:"-"`
	LeaseExpires int    `json:"leaseExpires,omitempty" deep:"-"`
	Dequeues     int    `json:"dequeues,omitempty"`
}

func (e *MaintenanceManifestDocument) String() string {
	return encodeJSON(e)
}
