package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// FetchClient is the interface that the Admin Portal Frontend uses to gather
// information about clusters. It returns frontend-suitable data structures.
type FetchClient interface {
	ClusterOperators(context.Context) (*ClusterOperatorsInformation, error)
	Regions(context.Context) (RegionInfo, error)
}

// client is an implementation of FetchClient. It currently contains a "fetcher"
// which is responsible for fetching information from the k8s clusters. The
// mechanism of fetching the data from the cluster and returning it to the
// frontend is deliberately split since in the future this fetcher will instead
// operate off a queue-like interface, similar to the RP Cluster backend. The
// client will then be responsible for caching and access control.
type client struct {
	log     *logrus.Entry
	cluster *api.OpenShiftClusterDocument
	fetcher *realFetcher
}

// realFetcher is responsible for fetching information from k8s clusters. It
// contains Kubernetes clients and returns the frontend-suitable data
// structures. The concrete implementation of FetchClient wraps this.
type realFetcher struct {
	log       *logrus.Entry
	configcli configclient.Interface
}

func newRealFetcher(log *logrus.Entry, dialer proxy.Dialer, doc *api.OpenShiftClusterDocument) (*realFetcher, error) {
	restConfig, err := restconfig.RestConfig(dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	configcli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &realFetcher{
		log:       log,
		configcli: configcli,
	}, nil
}

func NewFetchClient(log *logrus.Entry, dialer proxy.Dialer, cluster *api.OpenShiftClusterDocument) (FetchClient, error) {
	fetcher, err := newRealFetcher(log, dialer, cluster)
	if err != nil {
		return nil, err
	}

	return &client{
		log:     log,
		cluster: cluster,
		fetcher: fetcher,
	}, nil
}
