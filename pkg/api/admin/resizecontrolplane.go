package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type ResizeControlPlaneOperationStatus string

const (
	ResizeControlPlaneOperationStatusSucceeded ResizeControlPlaneOperationStatus = "Succeeded"
	ResizeControlPlaneOperationStatusFailed    ResizeControlPlaneOperationStatus = "Failed"
	ResizeControlPlaneOperationStatusSkipped   ResizeControlPlaneOperationStatus = "Skipped"
)

// ResizeControlPlaneResponse captures the outcome of the RP-backed control plane resize operation.
type ResizeControlPlaneResponse struct {
	Message      string                            `json:"message,omitempty"`
	ResourceID   string                            `json:"resourceId,omitempty"`
	VMSize       string                            `json:"vmSize,omitempty"`
	DeallocateVM bool                              `json:"deallocateVM,omitempty"`
	DurationMS   int64                             `json:"durationMs,omitempty"`
	Summary      ResizeControlPlaneSummary         `json:"summary,omitempty"`
	Phases       []ResizeControlPlanePhase         `json:"phases,omitempty"`
	Nodes        []ResizeControlPlaneNodeOperation `json:"nodes,omitempty"`
}

// ResizeControlPlaneSummary captures the high-level result of the operation.
type ResizeControlPlaneSummary struct {
	TotalNodes     int      `json:"totalNodes,omitempty"`
	NodesResized   int      `json:"nodesResized,omitempty"`
	NodesSkipped   int      `json:"nodesSkipped,omitempty"`
	ExecutionOrder []string `json:"executionOrder,omitempty"`
}

// ResizeControlPlanePhase captures a major phase of the resize flow.
type ResizeControlPlanePhase struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs,omitempty"`
	Message    string                            `json:"message,omitempty"`
	Checks     []ResizeControlPlaneCheck         `json:"checks,omitempty"`
}

// ResizeControlPlaneCheck captures an individual validation check inside pre-flight validation.
type ResizeControlPlaneCheck struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs,omitempty"`
	Message    string                            `json:"message,omitempty"`
}

// ResizeControlPlaneNodeOperation captures what happened for a specific control plane node.
type ResizeControlPlaneNodeOperation struct {
	Name         string                            `json:"name,omitempty"`
	SourceVMSize string                            `json:"sourceVmSize,omitempty"`
	TargetVMSize string                            `json:"targetVmSize,omitempty"`
	Status       ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS   int64                             `json:"durationMs,omitempty"`
	Message      string                            `json:"message,omitempty"`
	Steps        []ResizeControlPlaneStep          `json:"steps,omitempty"`
}

// ResizeControlPlaneStep captures a single step within a node resize operation.
type ResizeControlPlaneStep struct {
	Name       string                            `json:"name,omitempty"`
	Status     ResizeControlPlaneOperationStatus `json:"status,omitempty"`
	DurationMS int64                             `json:"durationMs,omitempty"`
	Message    string                            `json:"message,omitempty"`
}
