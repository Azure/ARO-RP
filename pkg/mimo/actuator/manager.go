package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
)

type Actuator interface {
	Process(context.Context, *api.MaintenanceManifestDocument, *api.OpenShiftClusterDocument) (bool, error)
	AddTask(string, TaskFunc)
}

type actuator struct {
	env        env.Interface
	log        *logrus.Entry
	restConfig *rest.Config
	now        func() time.Time

	mmf database.MaintenanceManifests

	tasks map[string]TaskFunc
}

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	restConfig *rest.Config,
	mmf database.MaintenanceManifests) (Actuator, error) {
	a := &actuator{
		env:        _env,
		restConfig: restConfig,
		log:        log,
		mmf:        mmf,
		tasks:      make(map[string]TaskFunc),

		now: time.Now,
	}

	return a, nil
}

func (a *actuator) AddTask(u string, t TaskFunc) {
	a.tasks[u] = t
}

func (a *actuator) Process(ctx context.Context, doc *api.MaintenanceManifestDocument, oc *api.OpenShiftClusterDocument) (bool, error) {
	var err error
	evaluationTime := a.now()

	if evaluationTime.After(time.Unix(int64(doc.MaintenanceManifest.RunBefore), 0)) {
		// timed out, mark as such
		a.log.Infof("marking %v as outdated: %v older than %v", doc.ID, doc.MaintenanceManifest.RunBefore, evaluationTime.UTC())

		_, err := a.mmf.EndLease(ctx, doc.ClusterID, doc.ID, api.MaintenanceManifestStateTimedOut, to.StringPtr(fmt.Sprintf("timed out at %s", evaluationTime.UTC())))
		return false, err
	}

	// here
	f, ok := a.tasks[doc.MaintenanceManifest.MaintenanceSetID]
	if !ok {
		a.log.Infof("not found %v", doc.MaintenanceManifest.MaintenanceSetID)
	}

	handler := &th{
		env: a.env,
	}
	state, msg := f(ctx, handler, doc, oc)
	_, err = a.mmf.EndLease(ctx, doc.ClusterID, doc.ID, state, &msg)
	if err != nil {
		a.log.Error(err)
	}

	return true, err
}
