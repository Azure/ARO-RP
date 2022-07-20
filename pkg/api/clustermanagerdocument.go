package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// OpenShiftClusterManagerConfigurationDocument represents OpenShift cluster manager configuration documents.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterManagerConfigurationDocuments struct {
	Count                                            int                                                `json:"_count,omitempty"`
	ResourceID                                       string                                             `json:"_rid,omitempty"`
	OpenShiftClusterManagementConfigurationDocuments []*OpenShiftClusterManagementConfigurationDocument `json:"Documents,omitempty"`
}

// String returns a JSON representation of the OpenShiftClusterManagerConfigurationDocuments struct.
func (c *OpenShiftClusterManagerConfigurationDocuments) String() string {
	return encodeJSON(c)
}

// OpenShiftClusterManagementConfigurationDocument represents an OpenShift cluster manager configuration document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterManagerConfigurationDocument struct {
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

	OpenShiftClusterManagementConfiguration *OpenShiftClusterManagementConfiguration `json:"openShiftClusterManagementConfiguration,omitempty"`
}
