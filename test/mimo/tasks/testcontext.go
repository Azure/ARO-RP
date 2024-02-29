package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/mimo/tasks"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

func NewFakeTestContext(env env.Interface, log *logrus.Entry, ch clienthelper.Interface) tasks.TaskContext {
	return &fakeTestContext{
		env: env,
		log: log,
		ch:  ch,
	}
}

type fakeTestContext struct {
	env env.Interface
	ch  clienthelper.Interface
	log *logrus.Entry
}

func (t *fakeTestContext) Environment() env.Interface {
	return t.env
}

func (t *fakeTestContext) ClientHelper() (clienthelper.Interface, error) {
	return t.ch, nil
}

func (t *fakeTestContext) Log() *logrus.Entry {
	return t.log
}
