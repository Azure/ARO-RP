package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type PoolWorkerType string

const (
	PoolWorkerTypeMonitor       PoolWorkerType = "monitor"
	PoolWorkerTypeMIMOActuator  PoolWorkerType = "mimo-actuator"
	PoolWorkerTypeMIMOScheduler PoolWorkerType = "mimo-scheduler"
)

// PoolWorker represents a worker in a pool that distributes work via owning
// different OpenShiftCluster buckets
type PoolWorker struct {
	MissingFields

	Buckets []string `json:"buckets,omitempty"`
}
