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
	Message      string                            `json:"message,omitempty"`
	ResourceID   string                            `json:"resourceId,omitempty"`
	VMSize       string                            `json:"vmSize,omitempty"`
	DeallocateVM bool                              `json:"deallocateVM"`
	DurationMS   int64                             `json:"durationMs"`
	Summary      ResizeControlPlaneSummary         `json:"summary,omitempty"`
	Phases       []ResizeControlPlanePhase         `json:"phases,omitempty"`
	Nodes        []ResizeControlPlaneNodeOperation `json:"nodes,omitempty"`
}

type ResizeControlPlaneSummary struct {
	TotalNodes     int      `json:"totalNodes"`
	NodesResized   int      `json:"nodesResized"`
	NodesSkipped   int      `json:"nodesSkipped"`
	ExecutionOrder []string `json:"executionOrder,omitempty"`
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
	Steps        []ResizeControlPlaneStep          `json:"steps,omitempty"`
}

type ResizeControlPlaneStep struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs"`
	Message    string                            `json:"message,omitempty"`
}
