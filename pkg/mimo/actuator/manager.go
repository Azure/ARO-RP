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

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	utilmimo "github.com/Azure/ARO-RP/pkg/util/mimo"
)

const maxDequeueCount = 5

type Actuator interface {
	Process(context.Context) (bool, error)
	AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask)
}

type taskContextWithGetResult interface {
	utilmimo.TaskContext
	GetResultMessage() string
}

type actuator struct {
	env                      env.Interface
	log                      *logrus.Entry
	now                      func() time.Time
	taskRunTimeout           time.Duration
	manifestQueryBatchLength int

	clusterResourceID string

	sub database.Subscriptions
	oc  database.OpenShiftClusters
	mmf database.MaintenanceManifests

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask
}

var _ Actuator = (*actuator)(nil)

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	clusterResourceID string,
	sub database.Subscriptions,
	oc database.OpenShiftClusters,
	mmf database.MaintenanceManifests,
	now func() time.Time,
) (Actuator, error) {
	a := &actuator{
		env:                      _env,
		log:                      log,
		clusterResourceID:        strings.ToLower(clusterResourceID),
		sub:                      sub,
		oc:                       oc,
		mmf:                      mmf,
		tasks:                    make(map[api.MIMOTaskID]tasks.MaintenanceTask),
		taskRunTimeout:           time.Minute * 60,
		manifestQueryBatchLength: 50,

		now: now,
	}

	return a, nil
}

func (a *actuator) AddMaintenanceTasks(tasks map[api.MIMOTaskID]tasks.MaintenanceTask) {
	maps.Copy(a.tasks, tasks)
}

func (a *actuator) Process(ctx context.Context) (bool, error) {
	r, err := azure.ParseResourceID(a.clusterResourceID)
	if err != nil {
		err = fmt.Errorf("failed parsing ResourceID: %w", err)
		a.log.Error(err)
		return false, err
	}

	// Get the manifests for this cluster which need to be worked
	i, err := a.mmf.GetQueuedByClusterResourceID(ctx, a.clusterResourceID, "")
	if err != nil {
		err = fmt.Errorf("failed getting manifests: %w", err)
		a.log.Error(err)
		return false, err
	}

	docList := make([]*api.MaintenanceManifestDocument, 0)
	for {
		docs, err := i.Next(ctx, a.manifestQueryBatchLength)
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
		if evaluationTime.After(time.Unix(doc.MaintenanceManifest.RunBefore, 0)) {
			taskLog := a.log.WithFields(logrus.Fields{
				"manifestID": doc.ID,
				"taskID":     string(doc.MaintenanceManifest.MaintenanceTaskID),
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

	// Nothing to do, return early
	if len(manifestsToAction) == 0 {
		return false, nil
	}

	a.log.Infof("Processing %d manifests", len(manifestsToAction))

	// We need to fetch the subscription for the cluster to get the TenantID
	subDoc, err := a.sub.Get(ctx, strings.ToLower(r.SubscriptionID))
	if err != nil {
		err = fmt.Errorf("failed fetching subscription document: %w", err)
		a.log.Error(err)
		return false, err
	}

	doneSomeWork := false

	// Execute on the manifests we want to action
	for _, doc := range manifestsToAction {
		taskLog := a.log.WithFields(logrus.Fields{
			"manifestID": doc.ID,
			"taskID":     string(doc.MaintenanceManifest.MaintenanceTaskID),
		})
		taskLog.Info("begin processing manifest")

		// Fetch a fresh OpenShift cluster document, in case the previous task/a
		// concurrent action updated anything
		oc, err := a.oc.Get(ctx, a.clusterResourceID)
		if err != nil {
			taskLog.Errorf("failed fetching cluster document: %s", err.Error())
			return false, fmt.Errorf("failed getting cluster document: %w", err)
		}

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

		timeoutContext, cancel := context.WithTimeout(ctx, a.taskRunTimeout)

		// Create task context containing the environment, logger, cluster doc,
		// etc -- this is the only way we pass information, to reduce the
		// surface area for dependencies in tests
		taskContext := newTaskContext(timeoutContext, a.env, taskLog, oc, subDoc)

		// Perform the task with a timeout
		err = func() error {
			innerErr := f(taskContext, doc, oc)
			defer cancel()
			if innerErr != nil {
				return innerErr
			}
			return taskContext.Err()
		}()

		var state api.MaintenanceManifestState
		// Pull the result message out of the task context to save, if it is set
		msg := taskContext.(taskContextWithGetResult).GetResultMessage()

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

		doneSomeWork = true

		_, err = a.mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, state, &msg)
		if err != nil {
			taskLog.Error(fmt.Errorf("failed ending lease on manifest: %w", err))
		}
		taskLog.Info("manifest processing complete")
	}

	return doneSomeWork, nil
}
