package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
)

// Monitor represents a consistent interface for different monitoring components
type Monitor interface {
	Monitor(context.Context) error
	MonitorName() string
}

// noOpMonitor is a no operation monitor
type NoOpMonitor struct{}

func (no *NoOpMonitor) Monitor(context.Context) error {
	return nil
}

func (no *NoOpMonitor) MonitorName() string {
	return "noop"
}

// Closeable is implemented by monitors that hold resources requiring explicit
// cleanup.
// Close may be called during forced cleanup before Monitor has returned, so
// implementations must make it safe to call concurrently with Monitor and more
// than once.
type Closeable interface {
	Close()
}
