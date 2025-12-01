package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type MaintenanceManifestState string

const (
	MaintenanceManifestStatePending         MaintenanceManifestState = "Pending"
	MaintenanceManifestStateInProgress      MaintenanceManifestState = "InProgress"
	MaintenanceManifestStateCompleted       MaintenanceManifestState = "Completed"
	MaintenanceManifestStateFailed          MaintenanceManifestState = "Failed"
	MaintenanceManifestStateRetriesExceeded MaintenanceManifestState = "RetriesExceeded"
	MaintenanceManifestStateTimedOut        MaintenanceManifestState = "TimedOut"
	MaintenanceManifestStateCancelled       MaintenanceManifestState = "Cancelled"
)

type MIMOTaskID string

type MaintenanceManifest struct {
	// The ID for the resource.
	ID string `json:"id,omitempty"`

	ClusterResourceID string `json:"clusterResourceID,omitempty"`

	State      MaintenanceManifestState `json:"state,omitempty"`
	StatusText string                   `json:"statusText,omitempty"`

	MaintenanceTaskID MIMOTaskID `json:"maintenanceTaskID,omitempty"`
	Priority          int        `json:"priority,omitempty"`

	// RunAfter defines the earliest that this manifest should start running
	RunAfter int `json:"runAfter,omitempty"`
	// RunBefore defines the latest that this manifest should start running
	RunBefore int `json:"runBefore,omitempty"`
}

// MaintenanceManifestList represents a list of MaintenanceManifests.
type MaintenanceManifestList struct {
	// The list of MaintenanceManifests.
	MaintenanceManifests []*MaintenanceManifest `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}
