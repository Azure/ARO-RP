package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// FetchClient is the interface that the Admin Portal Frontend uses to gather
// information about clusters. It returns frontend-suitable data structures.
type FetchClient interface {
	Nodes(context.Context) (*NodeListInformation, error)
	ClusterOperators(context.Context) (*ClusterOperatorsInformation, error)
	Machines(context.Context) (*MachineListInformation, error)
	MachineSets(context.Context) (*MachineSetListInformation, error)
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
	log           *logrus.Entry
	configCli     configclient.Interface
	kubernetesCli kubernetes.Interface
	machineClient machineclient.Interface
}

func newRealFetcher(log *logrus.Entry, dialer proxy.Dialer, doc *api.OpenShiftClusterDocument) (*realFetcher, error) {
	restConfig, err := restconfig.RestConfig(dialer, doc.OpenShiftCluster)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	configCli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	kubernetesCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	machineClient, err := machineclient.NewForConfig(restConfig)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &realFetcher{
		log:           log,
		configCli:     configCli,
		kubernetesCli: kubernetesCli,
		machineClient: machineClient,
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
