package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// PlatformWorkloadIdentityRoleSetDocuments represents a set of PlatformWorkloadIdentityRoleSetDocuments.
// pkg/database/cosmosdb requires its definition.
type PlatformWorkloadIdentityRoleSetDocuments struct {
	Count                                    int                                        `json:"_count,omitempty"`
	ResourceID                               string                                     `json:"_rid,omitempty"`
	PlatformWorkloadIdentityRoleSetDocuments []*PlatformWorkloadIdentityRoleSetDocument `json:"Documents,omitempty"`
}

func (c *PlatformWorkloadIdentityRoleSetDocuments) String() string {
	return encodeJSON(c)
}

// PlatformWorkloadIdentityRoleSetDocument represents a document specifying a mapping from the names of OCP operators to the built-in roles that should be assigned to those operator's corresponding managed identities for a particular OCP version.
// pkg/database/cosmosdb requires its definition.
type PlatformWorkloadIdentityRoleSetDocument struct {
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

	PlatformWorkloadIdentityRoleSet *PlatformWorkloadIdentityRoleSet `json:"platformWorkloadIdentityRoleSet,omitempty"`
}

func (c *PlatformWorkloadIdentityRoleSetDocument) String() string {
	return encodeJSON(c)
}
