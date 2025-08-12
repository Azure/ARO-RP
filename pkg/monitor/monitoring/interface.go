package monitoring

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"
)

// Monitor represents a consistent interface for different monitoring components
type Monitor interface {
	Monitor(context.Context) error
}

// noOpMonitor is a no operation monitor
type NoOpMonitor struct {
	Wg *sync.WaitGroup
}

func (no *NoOpMonitor) Monitor(context.Context) error {
	no.Wg.Done()
	return nil
}
