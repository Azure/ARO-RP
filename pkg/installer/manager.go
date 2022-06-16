package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type manager struct {
	log *logrus.Entry
	env env.Interface

	doc             *api.OpenShiftClusterDocument
	subscriptionDoc *api.SubscriptionDocument
	fpAuthorizer    refreshable.Authorizer

	deployments features.DeploymentsClient

	graph graph.Manager

	kubernetescli kubernetes.Interface
}

type Interface interface {
	Install(ctx context.Context) error
}

func NewInstaller(log *logrus.Entry, _env env.Interface, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument, fpAuthorizer refreshable.Authorizer, deployments features.DeploymentsClient, g graph.Manager) Interface {
	return &manager{
		log:             log,
		env:             _env,
		doc:             doc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,
		deployments:     deployments,
		graph:           g,
	}
}
