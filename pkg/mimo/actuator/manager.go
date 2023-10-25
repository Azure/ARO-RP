package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
)

type Actuator interface {
	Process(context.Context, *api.OpenShiftClusterDocument) (bool, error)
	AddTask(string, TaskFunc)
}

type actuator struct {
	env        env.Interface
	log        *logrus.Entry
	restConfig *rest.Config
	now        func() time.Time

	oc  database.OpenShiftClusters
	mmf database.MaintenanceManifests

	tasks map[string]TaskFunc
}

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	restConfig *rest.Config,
	oc database.OpenShiftClusters,
	mmf database.MaintenanceManifests) (Actuator, error) {
	a := &actuator{
		env:        _env,
		restConfig: restConfig,
		log:        log,
		oc:         oc,
		mmf:        mmf,
		tasks:      make(map[string]TaskFunc),

		now: time.Now,
	}

	return a, nil
}

func (a *actuator) AddTask(u string, t TaskFunc) {
	a.tasks[u] = t
}

func (a *actuator) Process(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	var err error
	cID := strings.ToLower(doc.OpenShiftCluster.ID)

	release := func() {
		if doc.LeaseOwner == "" {
			return
		}
		doc, err = a.oc.EndLease(ctx, doc.ResourceID, doc.OpenShiftCluster.Properties.ProvisioningState, doc.OpenShiftCluster.Properties.LastProvisioningState, nil)
		if err != nil {
			a.log.Error(err)
		}
	}

	didWork := false

	for {
		task, err := a.mmf.Dequeue(ctx, cID)
		if err != nil {
			return didWork, err
		}

		if task == nil {
			break
		}

		// renew/get the lease on the OpenShiftClusterDocument
		doc, err = a.oc.Lease(ctx, cID)
		if err != nil {
			if err.Error() == "lost lease" {
				return false, nil
			}
			return false, err
		}
		defer release()

		evaluationTime := a.now()

		if evaluationTime.Before(time.Unix(int64(task.MaintenanceManifest.RunAfter), 0)) {
			continue
		}

		if evaluationTime.After(time.Unix(int64(task.MaintenanceManifest.RunBefore), 0)) {
			// timed out, mark as such
			a.log.Infof("marking %v as outdated: %v older than %v", task.ID, task.MaintenanceManifest.RunBefore, evaluationTime.UTC())

			_, err := a.mmf.EndLease(ctx, cID, task.ID, api.MaintenanceManifestStateTimedOut, to.StringPtr(fmt.Sprintf("timed out at %s", evaluationTime.UTC())))
			if err != nil {
				return false, err
			}
			continue
		}

		// here
		f, ok := a.tasks[task.MaintenanceManifest.MaintenanceSetID]
		if !ok {
			a.log.Infof("not found %v", task.MaintenanceManifest.MaintenanceSetID)
		}

		handler := &th{
			db:  a.oc,
			env: a.env,
		}
		state, msg := f(ctx, handler, doc, task)
		_, err = a.mmf.EndLease(ctx, cID, task.ID, state, &msg)
		if err != nil {
			a.log.Error(err)
		}

		didWork = true
	}

	return didWork, nil
}
