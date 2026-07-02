package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
)

type fakeScheduler struct {
	hasRan        bool
	whileRunning  func()
	waitOnProcess *sync.WaitGroup
}

func (f *fakeScheduler) AddMaintenanceTasks(_ map[api.MIMOTaskID]tasks.MaintenanceTask) {}
func (f *fakeScheduler) Process(_ context.Context) (bool, error) {
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
