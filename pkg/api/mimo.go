package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type MaintenanceManifestState string
type MaintenanceScheduleState string
type MaintenanceScheduleSelectorOperator string

const (
	MaintenanceManifestStatePending         MaintenanceManifestState = "Pending"
	MaintenanceManifestStateInProgress      MaintenanceManifestState = "InProgress"
	MaintenanceManifestStateCompleted       MaintenanceManifestState = "Completed"
	MaintenanceManifestStateFailed          MaintenanceManifestState = "Failed"
	MaintenanceManifestStateRetriesExceeded MaintenanceManifestState = "RetriesExceeded"
	MaintenanceManifestStateTimedOut        MaintenanceManifestState = "TimedOut"
	MaintenanceManifestStateCancelled       MaintenanceManifestState = "Cancelled"
)

const (
	MaintenanceScheduleStateEnabled    MaintenanceScheduleState = "Enabled"
	MaintenanceScheduleStateProcessing MaintenanceScheduleState = "Processing"
	MaintenanceManifestStateDisabled   MaintenanceScheduleState = "Disabled"
	MaintenanceManifestStateInvalid    MaintenanceScheduleState = "Invalid"
)

const (
	MaintenanceScheduleSelectorOperatorEq    MaintenanceScheduleSelectorOperator = "eq"
	MaintenanceScheduleSelectorOperatorIn    MaintenanceScheduleSelectorOperator = "in"
	MaintenanceScheduleSelectorOperatorNotIn MaintenanceScheduleSelectorOperator = "notin"
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

type MaintenanceSchedule struct {
	MissingFields

	State             MaintenanceScheduleState `json:"state,omitempty"`
	MaintenanceTaskID MIMOTaskID               `json:"maintenanceTaskID,omitempty"`

	Schedule         string `json:"schedule,omitempty"`
	LookForwardCount int    `json:"lookForwardCount,omitempty"`
	ScheduleAcross   string `json:"scheduleAcross,omitempty"`

	Selectors []*MaintenanceScheduleSelector
}

type MaintenanceScheduleSelector struct {
	Key      string                              `json:"key,omitempty"`
	Operator MaintenanceScheduleSelectorOperator `json:"operator,omitempty"`
	Value    string                              `json:"value,omitempty"`
	Values   []string                            `json:"values,omitempty"`
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
