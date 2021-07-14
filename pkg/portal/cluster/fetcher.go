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

type FetchClient interface {
	ClusterOperators(context.Context) (*ClusterOperatorsInformation, error)
}

type realFetcher struct {
	log       *logrus.Entry
	configcli configclient.Interface
}

type client struct {
	log     *logrus.Entry
	cluster *api.OpenShiftClusterDocument
	fetcher *realFetcher
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
