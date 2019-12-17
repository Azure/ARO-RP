package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	uuid "github.com/satori/go.uuid"
)

// OpenShiftClusterDocuments represents OpenShift cluster documents.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocuments struct {
	Count                     int                         `json:"_count,omitempty"`
	ResourceID                string                      `json:"_rid,omitempty"`
	OpenShiftClusterDocuments []*OpenShiftClusterDocument `json:"Documents,omitempty"`
}

// OpenShiftClusterDocument represents an OpenShift cluster document.
// pkg/database/cosmosdb requires its definition.
type OpenShiftClusterDocument struct {
	MissingFields

	ID          string `json:"id,omitempty"`
	ResourceID  string `json:"_rid,omitempty"`
	Timestamp   int    `json:"_ts,omitempty"`
	Self        string `json:"_self,omitempty"`
	ETag        string `json:"_etag,omitempty"`
	Attachments string `json:"_attachments,omitempty"`

	Key          Key    `json:"key,omitempty"`
	PartitionKey string `json:"partitionKey,omitempty"`

	LeaseOwner   *uuid.UUID `json:"leaseOwner,omitempty"`
	LeaseExpires int        `json:"leaseExpires,omitempty"`
	Dequeues     int        `json:"dequeues,omitempty"`

	OpenShiftCluster *OpenShiftCluster `json:"openShiftCluster,omitempty"`
}

// Key represents a database lookup key.
type Key string
