package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/service"
)

type actuator struct {
	env env.Interface
	log *logrus.Entry

	oc database.OpenShiftClusters
	q  service.Runnable
}

func NewActuator(
	ctx context.Context,
	_env env.Interface,
	log *logrus.Entry,
	oc database.OpenShiftClusters) (service.Runnable, error) {
	a := &actuator{
		env: _env,
		log: log,
		oc:  oc,
	}

	a.q = service.NewWorkerQueue(ctx, log, _env, a.try)
	return a.q, nil
}

func (a *actuator) try(ctx context.Context, c *sync.Cond) (bool, error) {
	return false, nil
}
