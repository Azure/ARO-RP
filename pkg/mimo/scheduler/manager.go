package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
)

type getCachedScheduleDocFunc func() (*api.MaintenanceScheduleDocument, bool)

// get the list of clusters that we have cached
type getClustersFunc func() iter.Seq2[string, selectorData]

type Scheduler interface {
	Process(context.Context) (bool, error)
	AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask)
}

type scheduler struct {
	env env.Interface
	log *logrus.Entry
	now func() time.Time

	cachedDoc   getCachedScheduleDocFunc
	getClusters getClustersFunc

	dbs schedulerDBs

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask
}

var _ Scheduler = (*scheduler)(nil)

func NewSchedulerForSchedule(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	cachedDoc getCachedScheduleDocFunc,
	getClusters getClustersFunc,
	dbs schedulerDBs,
	now func() time.Time) (Scheduler, error) {
	a := &scheduler{
		env: _env,
		log: log,

		cachedDoc:   cachedDoc,
		getClusters: getClusters,

		dbs:   dbs,
		tasks: make(map[api.MIMOTaskID]tasks.MaintenanceTask),

		now: now,
	}

	return a, nil
}

func (a *scheduler) AddMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	maps.Copy(a.tasks, tasks)
}

func (a *scheduler) Process(ctx context.Context) (bool, error) {
	doc, ok := a.cachedDoc()
	if !ok {
		a.log.Error("can't get the schedule doc?")
		return false, errors.New("can't get the cached schedule doc")
	}

	a.log.Infof("processing schedule %s", doc.ID)

	// go over each of the clusters
	for id, cl := range a.getClusters() {
		a.log.Infof("checking selectors for %s (sub %s)", id, cl.SubscriptionID)
	}

	return true, nil
}
