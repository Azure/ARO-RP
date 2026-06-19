package actuator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
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
	dbs actuatorDBs

	_ch                  clienthelper.Interface
	_aroOperatorDeployer deploy.Operator

	az *azClients
}

// force interface checking
var _ mimo.TaskContext = &th{}

func newTaskContext(ctx context.Context, env env.Interface, log *logrus.Entry, dbs actuatorDBs, oc *api.OpenShiftClusterDocument, sub *api.SubscriptionDocument) *th {
	return &th{
		ctx: ctx,
		env: env,
		log: log,
		oc:  oc,
		sub: sub,
		dbs: dbs,
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

func (t *th) AROOperatorDeployer() (deploy.Operator, error) {
	if t._aroOperatorDeployer != nil {
		return t._aroOperatorDeployer, nil
	}

	restConfig, err := restconfig.RestConfig(t.env, t.oc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	ch, err := t.ClientHelper()
	if err != nil {
		return nil, err
	}

	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	extensionscli, err := extensionsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	operatorcli, err := operatorclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	t._aroOperatorDeployer, err = deploy.New(t.log, t.env, t.oc.OpenShiftCluster, t.sub, arocli, ch, extensionscli, kubernetescli, operatorcli)
	if err != nil {
		return nil, err
	}

	return t._aroOperatorDeployer, nil
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

func (t *th) PatchOpenShiftClusterDocument(ctx context.Context, f database.OpenShiftClusterDocumentMutator) (*api.OpenShiftClusterDocument, error) {
	db, err := t.dbs.OpenShiftClusters()
	if err != nil {
		return nil, err
	}
	return db.Patch(ctx, t.oc.Key, f)
}

// GetOpenShiftClusterDocument implements mimo.TaskContext.
func (t *th) GetOpenShiftClusterDocument() *api.OpenShiftClusterDocument {
	return t.oc
}

// getResultMessage is used by the Actuator to retrieve the finished result
// message out of the TaskContext
func (t *th) getResultMessage() string {
	return t.resultMessage
}
