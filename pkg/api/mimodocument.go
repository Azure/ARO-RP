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

func (c *MaintenanceManifestDocuments) GetCount() int {
	if c == nil {
		return 0
	}
	return c.Count
}

func (c *MaintenanceManifestDocuments) Docs() []*MaintenanceManifestDocument {
	if c == nil {
		return []*MaintenanceManifestDocument{}
	}
	return c.MaintenanceManifestDocuments
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

	ClusterResourceID   string              `json:"clusterResourceID,omitempty"`
	MaintenanceManifest MaintenanceManifest `json:"maintenanceManifest,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty" deep:"-"`
	LeaseExpires int    `json:"leaseExpires,omitempty" deep:"-"`
	Dequeues     int    `json:"dequeues,omitempty"`
}

func (c *MaintenanceManifestDocument) GetID() string {
	return c.ID
}

func (e *MaintenanceManifestDocument) String() string {
	return encodeJSON(e)
}

type MaintenanceScheduleDocuments struct {
	Count                        int                            `json:"_count,omitempty"`
	ResourceID                   string                         `json:"_rid,omitempty"`
	MaintenanceScheduleDocuments []*MaintenanceScheduleDocument `json:"Documents,omitempty"`
}

func (e *MaintenanceScheduleDocuments) String() string {
	return encodeJSON(e)
}

type MaintenanceScheduleDocument struct {
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

	MaintenanceSchedule MaintenanceSchedule `json:"maintenanceSchedule,omitempty"`

	LeaseOwner   string `json:"leaseOwner,omitempty" deep:"-"`
	LeaseExpires int    `json:"leaseExpires,omitempty" deep:"-"`
	Dequeues     int    `json:"dequeues,omitempty"`
}

func (e *MaintenanceScheduleDocument) String() string {
	return encodeJSON(e)
}

func (c *MaintenanceScheduleDocument) GetID() string {
	return c.ID
}

func (c *MaintenanceScheduleDocument) GetKey() string {
	return c.ID
}

func (c *MaintenanceScheduleDocument) GetBucket() int {
	return 0
}
