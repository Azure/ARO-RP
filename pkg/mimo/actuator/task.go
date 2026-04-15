package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type th struct {
	ctx context.Context

	env env.Interface
	log *logrus.Entry

	resultMessage string

	oc  *api.OpenShiftClusterDocument
	sub *api.SubscriptionDocument

	_ch clienthelper.Interface

	az *azClients
}

// force interface checking
var _ mimo.TaskContext = &th{}

func newTaskContext(ctx context.Context, env env.Interface, log *logrus.Entry, oc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument) *th {
	return &th{
		ctx: ctx,
		env: env,
		log: log,
		oc:  oc,
		sub: sub,
		_ch: nil,
	}
}

// context stuff
func (t *th) Deadline() (time.Time, bool) {
	return t.ctx.Deadline()
}

func (t *th) Done() <-chan struct{} {
	return t.ctx.Done()
}

func (t *th) Err() error {
	return t.ctx.Err()
}

func (t *th) Value(key any) any {
	return t.ctx.Value(key)
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

	mapper, err := apiutil.NewDynamicRESTMapper(restConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		return nil, err
	}

	client, err := client.New(restConfig, client.Options{
		Mapper: mapper,
	})
	if err != nil {
		return nil, err
	}

	t._ch = clienthelper.NewWithClient(t.log, client)
	return t._ch, nil
}

func (t *th) Log() *logrus.Entry {
	return t.log
}

func (t *th) Now() time.Time {
	return time.Now()
}

func (t *th) SetResultMessage(msg string) {
	t.resultMessage = msg
}

func (t *th) GetClusterUUID() string {
	return t.oc.ID
}

func (t *th) GetOpenShiftClusterProperties() api.OpenShiftClusterProperties {
	return t.oc.OpenShiftCluster.Properties
}

// GetOpenshiftClusterDocument implements mimo.TaskContext.
func (t *th) GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument {
	return t.oc
}

// getResultMessage is used by the Actuator to retrieve the finished result
// message out of the TaskContext
func (t *th) getResultMessage() string {
	return t.resultMessage
}
