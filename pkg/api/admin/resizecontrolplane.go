package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type ResizeControlPlaneOperationStatus string

const (
	ResizeControlPlaneOperationStatusSucceeded ResizeControlPlaneOperationStatus = "Succeeded"
	ResizeControlPlaneOperationStatusFailed    ResizeControlPlaneOperationStatus = "Failed"
	ResizeControlPlaneOperationStatusSkipped   ResizeControlPlaneOperationStatus = "Skipped"
)

type ResizeControlPlaneResponse struct {
	Status       ResizeControlPlaneOperationStatus `json:"status"`
	Message      string                            `json:"message,omitempty"`
	ResourceID   string                            `json:"resourceId,omitempty"`
	VMSize       string                            `json:"vmSize,omitempty"`
	DeallocateVM bool                              `json:"deallocateVM"`
	DurationMS   int64                             `json:"durationMs"`
	Summary      ResizeControlPlaneSummary         `json:"summary,omitempty"`
	Preflight    ResizeControlPlanePreflight       `json:"preflight,omitempty"`
	FailedPhase  string                            `json:"failedPhase,omitempty"`
	FailedNode   string                            `json:"failedNode,omitempty"`
	FailedStep   string                            `json:"failedStep,omitempty"`
	NextAction   string                            `json:"nextAction,omitempty"`
	Nodes        []ResizeControlPlaneNodeOperation `json:"nodes,omitempty"`
	Phases       []ResizeControlPlanePhase         `json:"phases,omitempty"`
}

type ResizeControlPlaneSummary struct {
	TotalNodes     int      `json:"totalNodes"`
	NodesResized   int      `json:"nodesResized"`
	NodesSkipped   int      `json:"nodesSkipped"`
	ExecutionOrder []string `json:"executionOrder,omitempty"`
}

type ResizeControlPlanePreflight struct {
	Status       ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS   int64                             `json:"durationMs"`
	FailedChecks []ResizeControlPlaneCheck         `json:"failedChecks,omitempty"`
}

type ResizeControlPlanePhase struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs"`
	Message    string                            `json:"message,omitempty"`
	Checks     []ResizeControlPlaneCheck         `json:"checks,omitempty"`
}

type ResizeControlPlaneCheck struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs"`
	Message    string                            `json:"message,omitempty"`
}

type ResizeControlPlaneNodeOperation struct {
	Name         string                            `json:"name,omitempty"`
	SourceVMSize string                            `json:"sourceVmSize,omitempty"`
	TargetVMSize string                            `json:"targetVmSize,omitempty"`
	Status       ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS   int64                             `json:"durationMs"`
	Message      string                            `json:"message,omitempty"`
	FailedStep   string                            `json:"failedStep,omitempty"`
	NextAction   string                            `json:"nextAction,omitempty"`
	Steps        []ResizeControlPlaneStep          `json:"steps,omitempty"`
}

type ResizeControlPlaneStep struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs"`
	Message    string                            `json:"message,omitempty"`
}
