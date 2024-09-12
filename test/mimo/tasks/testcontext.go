package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/go-autorest/autorest"
)

type fakeTestContext struct {
	context.Context
	now func() time.Time
	env env.Interface
	ch  clienthelper.Interface
	log *logrus.Entry

	clusterUUID       string
	clusterResourceID string
	properties        api.OpenShiftClusterProperties

	resultMessage string
}

type Option func(*fakeTestContext)

func WithClientHelper(ch clienthelper.Interface) Option {
	return func(ftc *fakeTestContext) {
		ftc.ch = ch
	}
}

func WithOpenShiftClusterDocument(oc *api.OpenShiftClusterDocument) Option {
	return func(ftc *fakeTestContext) {
		ftc.clusterUUID = oc.ID
		ftc.clusterResourceID = oc.OpenShiftCluster.ID
		ftc.properties = oc.OpenShiftCluster.Properties
	}
}

func WithOpenShiftClusterProperties(uuid string, oc api.OpenShiftClusterProperties) Option {
	return func(ftc *fakeTestContext) {
		ftc.clusterUUID = uuid
		ftc.properties = oc
	}
}

func NewFakeTestContext(ctx context.Context, env env.Interface, log *logrus.Entry, now func() time.Time, o ...Option) *fakeTestContext {
	ftc := &fakeTestContext{
		Context: ctx,
		env:     env,
		log:     log,
		now:     now,
	}
	for _, i := range o {
		i(ftc)
	}
	return ftc
}

func (t *fakeTestContext) LocalFpAuthorizer() (autorest.Authorizer, error) {
	myAuthorizer := autorest.NullAuthorizer{}
	return myAuthorizer, nil
}
func (t *fakeTestContext) GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument {
	myCD := &api.OpenShiftClusterDocument{}
	return myCD
}

// handle

func (t *fakeTestContext) Environment() env.Interface {
	return t.env
}

func (t *fakeTestContext) ClientHelper() (clienthelper.Interface, error) {
	if t.ch == nil {
		return nil, fmt.Errorf("missing clienthelper")
	}
	return t.ch, nil
}

func (t *fakeTestContext) Log() *logrus.Entry {
	return t.log
}

func (t *fakeTestContext) Now() time.Time {
	return t.now()
}

// OpenShiftCluster
func (t *fakeTestContext) GetClusterUUID() string {
	if t.clusterUUID == "" {
		panic("didn't set up openshiftcluster in test")
	}
	return t.clusterUUID
}

func (t *fakeTestContext) GetOpenShiftClusterProperties() api.OpenShiftClusterProperties {
	if t.clusterUUID == "" {
		panic("didn't set up openshiftcluster in test")
	}
	return t.properties
}

func (t *fakeTestContext) SetResultMessage(s string) {
	t.resultMessage = s
}

func (t *fakeTestContext) GetResultMessage() string {
	return t.resultMessage
}
