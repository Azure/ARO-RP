package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
)

type fakeActuator struct {
	hasRan        bool
	whileRunning  func()
	waitOnProcess *sync.WaitGroup
}

var _ Actuator = &fakeActuator{}

func (f *fakeActuator) Process(ctx context.Context) (bool, error) {
	if f.hasRan {
		return false, nil
	}
	if f.whileRunning != nil {
		f.whileRunning()
	}
	f.hasRan = true
	f.waitOnProcess.Done()
	return true, nil
}

// no-op
func (f *fakeActuator) AddMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
}
