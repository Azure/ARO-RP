package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

func NewFakeTestContext(ctx context.Context, env env.Interface, log *logrus.Entry, now func() time.Time, ch clienthelper.Interface) mimo.TaskContext {
	return &fakeTestContext{
		Context: ctx,
		env:     env,
		log:     log,
		ch:      ch,
		now:     now,
	}
}

type fakeTestContext struct {
	context.Context
	now func() time.Time
	env env.Interface
	ch  clienthelper.Interface
	log *logrus.Entry

	resultMessage string
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

func (t *fakeTestContext) Now() time.Time {
	return t.now()
}

func (t *fakeTestContext) SetResultMessage(s string) {
	t.resultMessage = s
}

func (t *fakeTestContext) GetResultMessage() string {
	return t.resultMessage
}
