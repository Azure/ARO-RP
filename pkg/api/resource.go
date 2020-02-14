package api

import (
	"encoding/json"
	"time"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// ResourceType represents a resource type
type ResourceType string

// ProvisioningState constants
const (
	ResourceTypePEVNET ResourceType = "pe-vnet"
)

// ResourceState represents a resource state
type ResourceState int

// InstallPhase constants
const (
	ResourceStateNew ResourceState = iota
	ResourceStateAllocated
	ResourceStateExpired
)

// Resource represents a resource
type Resource struct {
	MissingFields

	ClusterID        string        `json:"clusterID,omitempty"`
	Type             ResourceType  `json:"type,omitempty"`
	State            ResourceState `json:"state,omitempty"`
	AllocationTime   time.Time     `json:"allocationTime,omitempty"`
	DeallocationTime time.Time     `json:"deallocationTime,omitempty"`

	Spec json.RawMessage `json:"spec,omitempty"`
}
