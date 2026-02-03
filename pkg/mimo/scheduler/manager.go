package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
)

type getCachedDoc func() (*api.MaintenanceScheduleDocument, bool)

type Scheduler interface {
	Process(context.Context) (bool, error)
	AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask)
}

type scheduler struct {
	env env.Interface
	log *logrus.Entry
	now func() time.Time

	cachedDoc getCachedDoc

	dbs schedulerDBs

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask
}

var _ Scheduler = (*scheduler)(nil)

func NewSchedulerForSchedule(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	cachedDoc getCachedDoc,
	dbs schedulerDBs,
	now func() time.Time) (Scheduler, error) {
	a := &scheduler{
		env:       _env,
		log:       log,
		cachedDoc: cachedDoc,
		dbs:       dbs,
		tasks:     make(map[api.MIMOTaskID]tasks.MaintenanceTask),

		now: now,
	}

	return a, nil
}

func (a *scheduler) AddMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	maps.Copy(a.tasks, tasks)
}

func (a *scheduler) Process(ctx context.Context) (bool, error) {
	// // Get the manifests for this cluster which need to be worked
	// i, err := a.mmf.GetQueuedByClusterResourceID(ctx, a.clusterResourceID, "")
	// if err != nil {
	// 	err = fmt.Errorf("failed getting manifests: %w", err)
	// 	a.log.Error(err)
	// 	return false, err
	// }

	// docList := make([]*api.MaintenanceManifestDocument, 0)
	// for {
	// 	docs, err := i.Next(ctx, -1)
	// 	if err != nil {
	// 		err = fmt.Errorf("failed reading next manifest document: %w", err)
	// 		a.log.Error(err)
	// 		return false, err
	// 	}
	// 	if docs == nil {
	// 		break
	// 	}

	// 	docList = append(docList, docs.MaintenanceManifestDocuments...)
	// }
	return true, nil
}
