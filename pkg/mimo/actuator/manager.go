package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	utilmimo "github.com/Azure/ARO-RP/pkg/util/mimo"
)

const maxDequeueCount = 5

type Actuator interface {
	Process(context.Context) (bool, error)
	AddMaintenanceTasks(map[string]tasks.MaintenanceTask)
}

type actuator struct {
	env env.Interface
	log *logrus.Entry
	now func() time.Time

	clusterResourceID string

	oc  database.OpenShiftClusters
	mmf database.MaintenanceManifests

	tasks map[string]tasks.MaintenanceTask
}

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	clusterResourceID string,
	oc database.OpenShiftClusters,
	mmf database.MaintenanceManifests,
	now func() time.Time) (Actuator, error) {
	a := &actuator{
		env:               _env,
		log:               log,
		clusterResourceID: strings.ToLower(clusterResourceID),
		oc:                oc,
		mmf:               mmf,
		tasks:             make(map[string]tasks.MaintenanceTask),

		now: now,
	}

	return a, nil
}

func (a *actuator) AddMaintenanceTasks(tasks map[string]tasks.MaintenanceTask) {
	maps.Copy(a.tasks, tasks)
}

func (a *actuator) Process(ctx context.Context) (bool, error) {
	// Get the manifests for this cluster which need to be worked
	i, err := a.mmf.GetQueuedByClusterResourceID(ctx, a.clusterResourceID, "")
	if err != nil {
		err = fmt.Errorf("failed getting manifests: %w", err)
		a.log.Error(err)
		return false, err
	}

	docList := make([]*api.MaintenanceManifestDocument, 0)
	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			err = fmt.Errorf("failed reading next manifest document: %w", err)
			a.log.Error(err)
			return false, err
		}
		if docs == nil {
			break
		}

		docList = append(docList, docs.MaintenanceManifestDocuments...)
	}

	manifestsToAction := make([]*api.MaintenanceManifestDocument, 0)

	// Order manifests in order of RunAfter, and then Priority for ones with the
	// same RunAfter.
	sort.SliceStable(docList, func(i, j int) bool {
		if docList[i].MaintenanceManifest.RunAfter == docList[j].MaintenanceManifest.RunAfter {
			return docList[i].MaintenanceManifest.Priority < docList[j].MaintenanceManifest.Priority
		}

		return docList[i].MaintenanceManifest.RunAfter < docList[j].MaintenanceManifest.RunAfter
	})

	evaluationTime := a.now()

	// Check for manifests that have timed out first
	for _, doc := range docList {
		if evaluationTime.After(time.Unix(int64(doc.MaintenanceManifest.RunBefore), 0)) {
			taskLog := a.log.WithFields(logrus.Fields{
				"manifestID": doc.ID,
				"taskID":     doc.MaintenanceManifest.MaintenanceTaskID,
			})
			// timed out, mark as such
			taskLog.Infof("marking as outdated: %v older than %v", doc.MaintenanceManifest.RunBefore, evaluationTime.UTC())

			_, err := a.mmf.Patch(ctx, a.clusterResourceID, doc.ID, func(d *api.MaintenanceManifestDocument) error {
				d.MaintenanceManifest.State = api.MaintenanceManifestStateTimedOut
				d.MaintenanceManifest.StatusText = fmt.Sprintf("timed out at %s", evaluationTime.UTC())
				return nil
			})
			if err != nil {
				taskLog.Error(fmt.Errorf("failed to patch manifest with state TimedOut; will still attempt to process other manifests: %w", err))
			}
		} else {
			// not timed out, do something about it
			manifestsToAction = append(manifestsToAction, doc)
		}
	}

	// Nothing to do, don't dequeue
	if len(manifestsToAction) == 0 {
		return false, nil
	}

	a.log.Infof("Processing %d manifests", len(manifestsToAction))

	// Dequeue the document
	oc, err := a.oc.Get(ctx, a.clusterResourceID)
	if err != nil {
		return false, fmt.Errorf("failed getting cluster document: %w", err)
	}

	oc, err = a.oc.DoDequeue(ctx, oc)
	if err != nil {
		return false, fmt.Errorf("failed dequeuing cluster document: %w", err) // This will include StatusPreconditionFaileds
	}

	// Mark the maintenance state as unplanned and put it in AdminUpdating
	a.log.Infof("Marking cluster as in AdminUpdating")
	oc, err = a.oc.PatchWithLease(ctx, a.clusterResourceID, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.LastProvisioningState = oscd.OpenShiftCluster.Properties.ProvisioningState
		oscd.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
		oscd.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
		return nil
	})
	if err != nil {
		err = fmt.Errorf("failed setting provisioning state on cluster document: %w", err)
		a.log.Error(err)

		// attempt to dequeue the document, for what it's worth
		_, leaseErr := a.oc.EndLease(ctx, a.clusterResourceID, oc.OpenShiftCluster.Properties.LastProvisioningState, oc.OpenShiftCluster.Properties.FailedProvisioningState, nil)
		if leaseErr != nil {
			return false, fmt.Errorf("failed ending lease early on cluster document: %w", leaseErr)
		}
		return false, err
	}

	// Execute on the manifests we want to action
	for _, doc := range manifestsToAction {
		taskLog := a.log.WithFields(logrus.Fields{
			"manifestID": doc.ID,
			"taskID":     doc.MaintenanceManifest.MaintenanceTaskID,
		})
		taskLog.Info("begin processing manifest")

		// Attempt a dequeue
		doc, err = a.mmf.Lease(ctx, a.clusterResourceID, doc.ID)
		if err != nil {
			// log and continue to the next task if it doesn't work
			taskLog.Error(err)
			continue
		}

		// error if we don't know what this task is, then continue
		f, ok := a.tasks[doc.MaintenanceManifest.MaintenanceTaskID]
		if !ok {
			taskLog.Errorf("task %v not found", doc.MaintenanceManifest.MaintenanceTaskID)
			msg := "task ID not registered"
			_, err = a.mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, api.MaintenanceManifestStateFailed, &msg)
			if err != nil {
				taskLog.Error(fmt.Errorf("failed ending lease early on manifest: %w", err))
			}
			continue
		}

		taskLog.Info("executing manifest")

		// Create task context containing the environment, logger, cluster doc,
		// etc -- this is the only way we pass information, to reduce the
		// surface area for dependencies in tests
		taskContext := newTaskContext(ctx, a.env, taskLog, oc)

		// Perform the task with a timeout
		err = taskContext.RunInTimeout(time.Minute*60, func() error {
			innerErr := f(taskContext, doc, oc)
			if innerErr != nil {
				return innerErr
			}
			return taskContext.Err()
		})

		var state api.MaintenanceManifestState
		// Pull the result message out of the task context to save, if it is set
		msg := taskContext.GetResultMessage()

		if err != nil {
			if doc.Dequeues >= maxDequeueCount {
				msg = fmt.Sprintf("did not succeed after %d times, failing -- %s", doc.Dequeues, err.Error())
				state = api.MaintenanceManifestStateRetriesExceeded
				taskLog.Error(msg)
			} else if utilmimo.IsRetryableError(err) {
				// If an error is retryable (i.e explicitly marked as a transient error
				// by wrapping it in utilmimo.TransientError), then mark it back as
				// Pending so that it will get picked up and retried.
				state = api.MaintenanceManifestStatePending
				taskLog.Error(fmt.Errorf("task returned a retryable error: %w", err))
			} else {
				// Terminal errors (explicitly marked or unwrapped) cause task failure
				state = api.MaintenanceManifestStateFailed
				taskLog.Error(fmt.Errorf("task returned a terminal error: %w", err))
			}
		} else {
			// Mark tasks that don't have an error as succeeded implicitly
			state = api.MaintenanceManifestStateCompleted
			taskLog.Info("manifest executed successfully")
		}

		_, err = a.mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, state, &msg)
		if err != nil {
			taskLog.Error(fmt.Errorf("failed ending lease on manifest: %w", err))
		}
		taskLog.Info("manifest processing complete")
	}

	// Remove any set maintenance state
	a.log.Info("removing maintenance state on cluster")
	oc, err = a.oc.PatchWithLease(ctx, a.clusterResourceID, func(oscd *api.OpenShiftClusterDocument) error {
		oscd.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
		return nil
	})
	if err != nil {
		a.log.Error(fmt.Errorf("failed removing maintenance state on cluster document, but continuing: %w", err))
	}

	// release the OpenShiftCluster
	a.log.Info("ending lease on cluster")
	_, err = a.oc.EndLease(ctx, a.clusterResourceID, oc.OpenShiftCluster.Properties.LastProvisioningState, oc.OpenShiftCluster.Properties.FailedProvisioningState, nil)
	if err != nil {
		return false, fmt.Errorf("failed ending lease on cluster document: %w", err)
	}
	return true, nil
}
