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

	// clusterUUID is the UUID of the OpenShiftClusterDocument that contained
	// this OpenShiftCluster. It should be used where a unique ID for this
	// cluster is required.
	clusterUUID  string
	oc           *api.OpenShiftCluster
	sub          *api.Subscription
	fpAuthorizer refreshable.Authorizer

	deployments features.DeploymentsClient

	graph graph.Manager

	kubernetescli kubernetes.Interface
}

type Interface interface {
	Install(ctx context.Context) error
}

func NewInstaller(log *logrus.Entry, _env env.Interface, clusterUUID string, oc *api.OpenShiftCluster, subscription *api.Subscription, fpAuthorizer refreshable.Authorizer, deployments features.DeploymentsClient, g graph.Manager) Interface {
	return &manager{
		log:          log,
		env:          _env,
		clusterUUID:  clusterUUID,
		oc:           oc,
		sub:          subscription,
		fpAuthorizer: fpAuthorizer,
		deployments:  deployments,
		graph:        g,
	}
}
