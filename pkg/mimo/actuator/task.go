package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type th struct {
	env env.Interface
	log *logrus.Entry

	oc *api.OpenShiftClusterDocument

	_ch clienthelper.Interface
}

func newTaskContext(env env.Interface, log *logrus.Entry, oc *api.OpenShiftClusterDocument) tasks.TaskContext {
	return &th{
		env: env,
		log: log,
		oc:  oc,
		_ch: nil,
	}
}

func (t *th) Environment() env.Interface {
	return t.env
}

func (t *th) ClientHelper() (clienthelper.Interface, error) {
	if t._ch != nil {
		return t._ch, nil
	}

	restConfig, err := restconfig.RestConfig(t.env, t.oc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	ch, err := clienthelper.New(t.log, restConfig)
	if err != nil {
		return nil, err
	}

	t._ch = ch
	return t._ch, nil
}

func (t *th) Log() *logrus.Entry {
	return t.log
}
