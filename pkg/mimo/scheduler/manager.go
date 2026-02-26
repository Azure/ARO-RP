package scheduler

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"iter"
	"maps"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/monitor/dimension"
	"github.com/Azure/ARO-RP/pkg/util/log"
)

const friendlyDateFormat string = "2006-01-02T15:04Z07:00"

type newSchedulerFunc func(
	_env env.Interface,
	log *logrus.Entry,
	m metrics.Emitter,
	cachedDoc getCachedScheduleDocFunc,
	getClusters getClustersFunc,
	dbs schedulerDBs,
	now func() time.Time,
) (Scheduler, error)
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
	m   metrics.Emitter

	cachedDoc   getCachedScheduleDocFunc
	getClusters getClustersFunc

	dbs schedulerDBs

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask
}

var _ Scheduler = (*scheduler)(nil)

func NewSchedulerForSchedule(
	_env env.Interface,
	log *logrus.Entry,
	m metrics.Emitter,
	cachedDoc getCachedScheduleDocFunc,
	getClusters getClustersFunc,
	dbs schedulerDBs,
	now func() time.Time,
) (Scheduler, error) {
	a := &scheduler{
		env: _env,
		log: log,
		m:   m,

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
	manifestsDB, err := a.dbs.MaintenanceManifests()
	if err != nil {
		return false, fmt.Errorf("unable to get maintenancemanifests: %w", err)
	}

	doc, ok := a.cachedDoc()
	if !ok {
		return false, errors.New("can't get the cached schedule doc")
	}

	a.log.Infof("processing schedule %s (task ID=%s)", doc.ID, doc.MaintenanceSchedule.MaintenanceTaskID)

	scheduleWithin, err := time.ParseDuration(doc.MaintenanceSchedule.ScheduleAcross)
	if err != nil {
		a.log.Errorf("unrecognised scheduleacross: %s", err.Error())
		return false, err
	}

	calDef, err := ParseCalendar(doc.MaintenanceSchedule.Schedule)
	if err != nil {
		return false, err
	}
	now := a.now()

	next, hasFutureTime := Next(now, calDef)
	if !hasFutureTime {
		a.log.Warnf("schedule '%s' will never trigger again, skipping", doc.MaintenanceSchedule.Schedule)
		return true, nil
	}
	periods_friendly := []string{next.Format(friendlyDateFormat)}
	periods := []time.Time{next}

	if doc.MaintenanceSchedule.LookForwardCount > 1 {
		for i := range doc.MaintenanceSchedule.LookForwardCount - 1 {
			n, inFuture := Next(periods[len(periods)-1], calDef)
			if !inFuture {
				a.log.Infof("schedule '%s' will only trigger %d times but look forward is %d", doc.MaintenanceSchedule.Schedule, i+1, doc.MaintenanceSchedule.LookForwardCount)
				break
			}

			periods = append(periods, n)
			periods_friendly = append(periods_friendly, n.Format(friendlyDateFormat))
		}
	}

	a.log.Infof("next valid scheduled times: %s", strings.Join(periods_friendly, ", "))

	// go over each of the clusters
	for clusterID, cl := range a.getClusters() {
		a.log.Debugf("checking selectors for %s (sub %s)", clusterID, cl["subscriptionID"])
		clusterLog := log.EnrichWithResourceID(a.log, clusterID)

		matchesSelectors, err := cl.Matches(clusterLog, doc.MaintenanceSchedule.Selectors)
		if err != nil {
			clusterLog.Errorf("error matching selectors, skipping cluster: %s", err.Error())
			continue
		}

		if !matchesSelectors {
			clusterLog.Debugf("cluster does not match selectors")
			continue
		}

		clusterLog.Debugf("cluster matches selectors")

		// this is the amount of time we will be offset inside the
		// 'scheduleAcross' window.
		offsetWithinScheduleAcross := PercentWithinPeriod(ClusterResourceIDHashToScheduleWithinPercent(clusterID), scheduleWithin)

		clusterLog.Debugf("Calculated scheduleAcross offset is %s", offsetWithinScheduleAcross.String())

		foundPeriods := map[int64]string{}

		existingTasks, err := manifestsDB.GetFutureTasksForClusterAndScheduleID(ctx, clusterID, doc.ID, "")
		if err != nil {
			clusterLog.Errorf("unable to list future tasks for cluster: %s", err.Error())
			continue
		}

		success := false
		for {
			docs, err := existingTasks.Next(ctx, 10)
			if err != nil {
				clusterLog.Errorf("error when consuming matching tasks for cluster: %s", err.Error())
				break
			}
			if docs.GetCount() == 0 {
				success = true
				break
			}

			for _, d := range docs.MaintenanceManifestDocuments {
				targetTime := time.Unix(d.MaintenanceManifest.RunAfter, 0).Unix()
				foundPeriods[targetTime] = d.ID
			}
		}
		if !success {
			// we errored, so exit out
			continue
		}

		manifestsFound := 0
		manifestsCreated := 0
		manifestsCancelled := 0
		for _, target := range periods {
			targetWithOffset := target.Add(offsetWithinScheduleAcross)
			scheduleMatch, found := foundPeriods[targetWithOffset.Unix()]
			if !found {
				clusterLog.Debugf("creating manifest for %s window (%s)", target, targetWithOffset)

				newManifest, err := manifestsDB.Create(ctx, &api.MaintenanceManifestDocument{
					ID:                manifestsDB.NewUUID(),
					ClusterResourceID: clusterID,
					MaintenanceManifest: api.MaintenanceManifest{
						State: api.MaintenanceManifestStatePending,

						MaintenanceTaskID: doc.MaintenanceSchedule.MaintenanceTaskID,
						CreatedBySchedule: api.MIMOScheduleID(doc.ID),
						RunAfter:          targetWithOffset.Unix(),
						RunBefore:         targetWithOffset.Add(time.Hour).Unix(),
					},
				})
				if err != nil {
					clusterLog.Errorf("error creating new maintenancemanifest, skipping: %s", err.Error())
					break
				}

				clusterLog.Infof("created new manifest id=%s for %s window (%s)", newManifest.ID, target.Format(friendlyDateFormat), targetWithOffset.Format(time.RFC3339))
				manifestsCreated += 1
			} else {
				clusterLog.Debugf("found manifest for %s (%s)", target, scheduleMatch)
				// Remove it from the foundPeriods, so that we can remove any
				// remainders (e.g. if we changed the schedule)
				delete(foundPeriods, targetWithOffset.Unix())
				manifestsFound += 1
			}
		}

		// Cancel the manifests which are not required which this
		for _, notNeededManifest := range foundPeriods {
			_, err = manifestsDB.Patch(ctx, clusterID, notNeededManifest, func(mmd *api.MaintenanceManifestDocument) error {
				mmd.MaintenanceManifest.State = api.MaintenanceManifestStateCancelled
				mmd.MaintenanceManifest.StatusText = "Cancelled by Scheduler as did not match current schedule settings"
				return nil
			})
			if err != nil {
				clusterLog.Errorf("error cancelling unneeded manifest: %s", err.Error())
			} else {
				manifestsCancelled += 1
				clusterLog.Debugf("cancelled unneeded manifest %s", notNeededManifest)
			}
		}

		clusterLog.Infof("created=%d, found valid=%d, cancelled=%d", manifestsCreated, manifestsFound, manifestsCancelled)

		// emit metrics when we have created or cancelled manifests
		resID, err := azure.ParseResourceID(clusterID)
		if err != nil {
			// this basically can't happen
			continue
		}

		clusterDims := map[string]string{
			dimension.ResourceID:           clusterID,
			dimension.SubscriptionID:       resID.SubscriptionID,
			dimension.ClusterResourceGroup: resID.ResourceGroup,
			dimension.ResourceName:         resID.ResourceName,
		}
		if manifestsCreated > 0 {
			a.m.EmitGauge("mimo.scheduler.manifests.created", int64(manifestsCreated), clusterDims)
		}
		if manifestsCancelled > 0 {
			a.m.EmitGauge("mimo.scheduler.manifests.cancelled", int64(manifestsCancelled), clusterDims)
		}
	}

	return true, nil
}

// Take a cluster resource ID and deterministically turn it into a % to be used
// for placing the cluster in the "scheduleAcross". For example, 0.0 will mean
// that it will be scheduled exactly at the schedule time, 1.0 will mean
// scheduled at schedule time + scheduleAcross.
func ClusterResourceIDHashToScheduleWithinPercent(resourceID string) float64 {
	sum := crc32.Checksum([]byte(strings.ToLower(resourceID)), crc32.IEEETable)
	out := float64(sum) / float64(^uint32(0))
	return out
}

// Given a period and a float from 0.0-1.0, calculate the target time within
// that duration rounded to the second.
func PercentWithinPeriod(percent float64, scheduleWithin time.Duration) time.Duration {
	percentIn := time.Duration(int64(float64(int64(scheduleWithin)) * percent))
	return percentIn.Round(time.Second)
}
