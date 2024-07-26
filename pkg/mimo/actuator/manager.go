package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/sets"
)

const maxDequeueCount = 5

type Actuator interface {
	Process(context.Context) (bool, error)
	AddMaintenanceSets(map[string]sets.MaintenanceSet)
}

type actuator struct {
	env env.Interface
	log *logrus.Entry
	now func() time.Time

	clusterResourceID string

	oc  database.OpenShiftClusters
	mmf database.MaintenanceManifests

	sets map[string]sets.MaintenanceSet
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
		sets:              make(map[string]sets.MaintenanceSet),

		now: now,
	}

	return a, nil
}

func (a *actuator) AddMaintenanceSets(sets map[string]sets.MaintenanceSet) {
	maps.Copy(a.sets, sets)
}

func (a *actuator) Process(ctx context.Context) (bool, error) {
	// Get the manifests for this cluster which need to be worked
	i, err := a.mmf.GetQueuedByClusterResourceID(ctx, a.clusterResourceID, "")
	if err != nil {
		return false, fmt.Errorf("failed getting manifests: %w", err)
	}

	docList := make([]*api.MaintenanceManifestDocument, 0)
	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return false, fmt.Errorf("failed reading next manifest document: %w", err)
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
			// timed out, mark as such
			a.log.Infof("marking %v as outdated: %v older than %v", doc.ID, doc.MaintenanceManifest.RunBefore, evaluationTime.UTC())

			_, err := a.mmf.Patch(ctx, a.clusterResourceID, doc.ID, func(d *api.MaintenanceManifestDocument) error {
				d.MaintenanceManifest.State = api.MaintenanceManifestStateTimedOut
				d.MaintenanceManifest.StatusText = fmt.Sprintf("timed out at %s", evaluationTime.UTC())
				return nil
			})
			if err != nil {
				a.log.Error(err)
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

	// Dequeue the document
	oc, err := a.oc.Get(ctx, a.clusterResourceID)
	if err != nil {
		return false, err
	}

	oc, err = a.oc.DoDequeue(ctx, oc)
	if err != nil {
		return false, err // This will include StatusPreconditionFaileds
	}

	taskContext := newTaskContext(ctx, a.env, a.log, oc)

	// Execute on the manifests we want to action
	for _, doc := range manifestsToAction {
		// here
		f, ok := a.sets[doc.MaintenanceManifest.MaintenanceSetID]
		if !ok {
			a.log.Infof("not found %v", doc.MaintenanceManifest.MaintenanceSetID)
			continue
		}

		// Attempt a dequeue
		doc, err = a.mmf.Lease(ctx, a.clusterResourceID, doc.ID)
		if err != nil {
			// log and continue to the next task if it doesn't work
			a.log.Error(err)
			continue
		}

		// if we've tried too many times, give up
		if doc.Dequeues > maxDequeueCount {
			err := fmt.Errorf("dequeued %d times, failing", doc.Dequeues)
			_, leaseErr := a.mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, api.MaintenanceManifestStateTimedOut, to.StringPtr(err.Error()))
			if leaseErr != nil {
				a.log.Error(err)
			}
			continue
		}

		var state api.MaintenanceManifestState
		var msg string

		// Perform the task with a timeout
		err = taskContext.RunInTimeout(time.Minute*60, func() error {
			state, msg = f(taskContext, doc, oc)
			return taskContext.Err()
		})
		if err != nil {
			a.log.Error(err)
		}

		_, err = a.mmf.EndLease(ctx, doc.ClusterResourceID, doc.ID, state, &msg)
		if err != nil {
			a.log.Error(err)
		}
	}

	// release the OpenShiftCluster
	_, err = a.oc.EndLease(ctx, a.clusterResourceID, oc.OpenShiftCluster.Properties.ProvisioningState, api.ProvisioningStateMaintenance, nil)
	return true, err
}
