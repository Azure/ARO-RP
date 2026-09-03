package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	utilmimo "github.com/Azure/ARO-RP/pkg/util/mimo"
)

var errFailedFetchingSubscriptionDocument = errors.New("failed fetching subscription document")

const maxDequeueCount = 5

type Actuator interface {
	Process(context.Context) (bool, error)
	AddMaintenanceTasks(map[api.MIMOTaskID]tasks.MaintenanceTask)
}

type newActuatorInstance = func(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	clusterResourceID string,
	dbs actuatorDBs,
) (Actuator, error)

type actuator struct {
	env                      env.Interface
	log                      *logrus.Entry
	taskRunTimeout           time.Duration
	manifestQueryBatchLength int

	clusterResourceID string

	dbs actuatorDBs

	tasks map[api.MIMOTaskID]tasks.MaintenanceTask
}

var _ Actuator = (*actuator)(nil)

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	clusterResourceID string,
	dbs actuatorDBs,
) (Actuator, error) {
	a := &actuator{
		env:                      _env,
		log:                      log,
		clusterResourceID:        strings.ToLower(clusterResourceID),
		dbs:                      dbs,
		tasks:                    make(map[api.MIMOTaskID]tasks.MaintenanceTask),
		taskRunTimeout:           time.Minute * 60,
		manifestQueryBatchLength: 50,
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
		return false, err
	}

	mmf, err := a.dbs.MaintenanceManifests()
	if err != nil {
		return false, err
	}

	ocDb, err := a.dbs.OpenShiftClusters()
	if err != nil {
		return false, err
	}

	subDb, err := a.dbs.Subscriptions()
	if err != nil {
		return false, err
	}

	// Get the manifests for this cluster which need to be worked
	i, err := mmf.GetQueuedByClusterResourceID(ctx, a.clusterResourceID, "")
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

	evaluationTime := a.env.Now()

	// Check for manifests that have timed out first
	for _, doc := range docList {
		if evaluationTime.After(time.Unix(doc.MaintenanceManifest.RunBefore, 0)) {
			taskLog := a.log.WithFields(logrus.Fields{
				"manifestID": doc.ID,
				"taskID":     string(doc.MaintenanceManifest.MaintenanceTaskID),
			})
			// timed out, mark as such
			taskLog.Infof("marking as outdated: %v older than %v", doc.MaintenanceManifest.RunBefore, evaluationTime.UTC())

			_, err := mmf.Patch(ctx, a.clusterResourceID, doc.ID, func(d *api.MaintenanceManifestDocument) error {
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
	subDoc, err := subDb.Get(ctx, strings.ToLower(r.SubscriptionID))
	if err != nil {
		err = fmt.Errorf("%w: %w", errFailedFetchingSubscriptionDocument, err)
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

		// Attempt a dequeue
		doc, err = mmf.Lease(ctx, a.clusterResourceID, doc.ID)
		if err != nil {
			// log and continue to the next task if it doesn't work
			taskLog.Error(err)
			continue
		}

		// Fetch a fresh OpenShift cluster document, in case the previous task/a
		// concurrent action updated anything.
		//
		// This is deliberately done after leasing. Lease is what increments
		// Dequeues, so fetching beforehand meant that a cluster document which
		// could not be read left the manifest's attempt count untouched, and
		// the manifest was retried indefinitely rather than eventually being
		// marked RetriesExceeded.
		oc, err := ocDb.Get(ctx, a.clusterResourceID)
		if err != nil {
			// Logged in the original wording, before wrapping, so that queries
			// and alerting over the actuator's logs are unaffected.
			taskLog.Errorf("failed fetching cluster document: %s", err.Error())

			// Marked transient so that a momentary failure to read the document
			// is retried rather than failing the manifest outright. A durable
			// one is still bounded, by maxDequeueCount.
			err = utilmimo.TransientError(fmt.Errorf("failed getting cluster document: %w", err))

			state, msg := manifestStateForError(doc.Dequeues, err.Error(), err)
			if _, endErr := mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, state, &msg); endErr != nil {
				taskLog.Error(fmt.Errorf("failed ending lease on manifest: %w", endErr))
			}

			// Continue rather than return: being unable to read this cluster's
			// document says nothing about the other manifests queued against
			// it, and returning here abandoned them for the whole run.
			doneSomeWork = true
			continue
		}

		// error if we don't know what this task is, then continue
		f, ok := a.tasks[doc.MaintenanceManifest.MaintenanceTaskID]
		if !ok {
			taskLog.Errorf("task %v not found", doc.MaintenanceManifest.MaintenanceTaskID)
			msg := "task ID not registered"
			_, err = mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, api.MaintenanceManifestStateFailed, &msg)
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
		taskContext := newTaskContext(timeoutContext, a.env, taskLog, a.dbs, oc, subDoc)

		// Perform the task with a timeout
		err = func() error {
			defer cancel()
			innerErr := f(taskContext, doc, oc)
			if innerErr != nil {
				return innerErr
			}
			return taskContext.Err()
		}()

		var state api.MaintenanceManifestState
		// Pull the result message out of the task context to save, if it is set
		msg := taskContext.getResultMessage()

		if err != nil {
			state, msg = manifestStateForError(doc.Dequeues, msg, err)

			switch state {
			case api.MaintenanceManifestStateRetriesExceeded:
				taskLog.Error(msg)
			case api.MaintenanceManifestStatePending:
				taskLog.Error(fmt.Errorf("task returned a retryable error: %w", err))
			default:
				taskLog.Error(fmt.Errorf("task returned a terminal error: %w", err))
			}
		} else {
			// Mark tasks that don't have an error as succeeded implicitly
			state = api.MaintenanceManifestStateCompleted
			taskLog.Info("manifest executed successfully")
		}

		doneSomeWork = true

		_, err = mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, state, &msg)
		if err != nil {
			taskLog.Error(fmt.Errorf("failed ending lease on manifest: %w", err))
		}
		taskLog.Info("manifest processing complete")
	}

	return doneSomeWork, nil
}

// manifestStateForError returns the state in which a manifest should be left
// after an attempt to action it returned err, together with the status text to
// record against it.
//
// Attempts are bounded by maxDequeueCount rather than by the nature of the
// error, so that a manifest which keeps failing transiently is eventually given
// up on rather than retried indefinitely. That bound relies on the manifest
// having been leased, since Lease is what increments Dequeues.
//
// RetriesExceeded is terminal: the dequeue query matches Pending only, so such
// a manifest is not picked up again even once whatever caused the failure has
// been resolved. The window it was scheduled for is therefore missed rather
// than deferred. This is bounded rather than permanent, because schedules are
// recurring and the scheduler creates a fresh manifest for the next window, but
// it does mean a durable fault costs a maintenance window per manifest.
func manifestStateForError(dequeues int, msg string, err error) (api.MaintenanceManifestState, string) {
	switch {
	case dequeues >= maxDequeueCount:
		return api.MaintenanceManifestStateRetriesExceeded,
			fmt.Sprintf("did not succeed after %d times, failing -- %s", dequeues, err.Error())
	case utilmimo.IsRetryableError(err):
		// An error explicitly marked as transient, by wrapping it in
		// utilmimo.TransientError, is marked back to Pending so that it is
		// picked up and retried.
		return api.MaintenanceManifestStatePending, msg
	default:
		// Terminal errors, whether explicitly marked or unwrapped, fail the
		// manifest.
		return api.MaintenanceManifestStateFailed, msg
	}
}
