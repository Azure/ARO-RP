package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type (
	MaintenanceManifestState            string
	MaintenanceScheduleState            string
	MaintenanceScheduleSelectorOperator string
)

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
	MaintenanceScheduleStateEnabled  MaintenanceScheduleState = "Enabled"
	MaintenanceScheduleStateDisabled MaintenanceScheduleState = "Disabled"
)

const (
	MaintenanceScheduleSelectorOperatorEq    MaintenanceScheduleSelectorOperator = "eq"
	MaintenanceScheduleSelectorOperatorIn    MaintenanceScheduleSelectorOperator = "in"
	MaintenanceScheduleSelectorOperatorNotIn MaintenanceScheduleSelectorOperator = "notin"
)

func validSelectorOperators() []MaintenanceScheduleSelectorOperator {
	return []MaintenanceScheduleSelectorOperator{
		MaintenanceScheduleSelectorOperatorEq,
		MaintenanceScheduleSelectorOperatorIn,
		MaintenanceScheduleSelectorOperatorNotIn,
	}
}

type (
	MIMOTaskID     string
	MIMOScheduleID string
)

type MaintenanceManifest struct {
	// The ID for the resource.
	ID string `json:"id,omitempty"`

	ClusterResourceID string `json:"clusterResourceID,omitempty"`

	State      MaintenanceManifestState `json:"state,omitempty"`
	StatusText string                   `json:"statusText,omitempty"`

	MaintenanceTaskID MIMOTaskID     `json:"maintenanceTaskID,omitempty"`
	CreatedBySchedule MIMOScheduleID `json:"createdBySchedule,omitempty"`
	Priority          int            `json:"priority,omitempty"`

	// RunAfter defines the earliest that this manifest should start running
	RunAfter int64 `json:"runAfter,omitempty"`
	// RunBefore defines the latest that this manifest should start running
	RunBefore int64 `json:"runBefore,omitempty"`
}

// MaintenanceManifestList represents a list of MaintenanceManifests.
type MaintenanceManifestList struct {
	// The list of MaintenanceManifests.
	MaintenanceManifests []*MaintenanceManifest `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}

type MaintenanceSchedule struct {
	ID string `json:"id,omitempty"`

	State             MaintenanceScheduleState `json:"state,omitempty" mutable:"true"`
	MaintenanceTaskID MIMOTaskID               `json:"maintenanceTaskID,omitempty"`

	Schedule         string `json:"schedule,omitempty" mutable:"true"`
	LookForwardCount int    `json:"lookForwardCount,omitempty" mutable:"true"`
	ScheduleAcross   string `json:"scheduleAcross,omitempty" mutable:"true"`

	Selectors []*MaintenanceScheduleSelector `json:"selectors,omitempty" mutable:"true"`
}

type MaintenanceScheduleSelector struct {
	Key      string                              `json:"key,omitempty"`
	Operator MaintenanceScheduleSelectorOperator `json:"operator,omitempty"`
	Value    string                              `json:"value,omitempty"`
	Values   []string                            `json:"values,omitempty"`
}

// MaintenanceScheduleList represents a list of MaintenanceSchedules.
type MaintenanceScheduleList struct {
	// The list of MaintenanceManifests.
	MaintenanceSchedules []*MaintenanceSchedule `json:"value"`

	// The link used to get the next page of operations.
	NextLink string `json:"nextLink,omitempty"`
}
