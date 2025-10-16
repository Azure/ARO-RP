package api

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

// MaintenanceManifest represents an instance of a MaintenanceTask running on a
// given cluster.
type MaintenanceManifest struct {
	MissingFields

	State      MaintenanceManifestState `json:"state,omitempty"`
	StatusText string                   `json:"statusText,omitempty"`

	MaintenanceTaskID MIMOTaskID `json:"maintenanceTaskID,omitempty"`
	Priority          int        `json:"priority,omitempty"`

	// RunAfter defines the earliest that this manifest should start running
	RunAfter int `json:"runAfter,omitempty"`
	// RunBefore defines the latest that this manifest should start running
	RunBefore int `json:"runBefore,omitempty"`
}
