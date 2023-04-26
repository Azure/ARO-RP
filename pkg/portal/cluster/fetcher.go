package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jongio/azidext/go/azidext"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type ResourceClientFactory interface {
	NewResourcesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) features.ResourcesClient
}

type VirtualMachinesClientFactory interface {
	NewVirtualMachinesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) compute.VirtualMachinesClient
}

// FetchClient is the interface that the Admin Portal Frontend uses to gather
// information about clusters. It returns frontend-suitable data structures.
type FetchClient interface {
	Nodes(context.Context) (*NodeListInformation, error)
	ClusterOperators(context.Context) (*ClusterOperatorsInformation, error)
	Machines(context.Context) (*MachineListInformation, error)
	MachineSets(context.Context) (*MachineSetListInformation, error)
	Statistics(context.Context, *http.Client, string, time.Duration, time.Time, string) ([]Metrics, error)
}

type AzureFetchClient interface {
	VMAllocationStatus(context.Context) (map[string]string, error)
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

// azureClient is the same implementation as client's, the only difference is that it will be used to fetch something from azure regarding a cluster.
type azureClient struct {
	log     *logrus.Entry
	fetcher *azureSideFetcher
}

// azureSideFetcher is responsible for fetching information about azure resources of a k8s cluster. It
// contains azure related authentication/authorization data and returns the frontend-suitable data
// structures. The concrete implementation of AzureFetchClient wraps this.
type azureSideFetcher struct {
	log                          *logrus.Entry
	resourceGroupName            string
	subscriptionDoc              *api.SubscriptionDocument
	spAuthorizer                 autorest.Authorizer
	env                          env.Core
	resourceClientFactory        ResourceClientFactory
	virtualMachinesClientFactory VirtualMachinesClientFactory
}

type clientFactory struct{}

func (cf clientFactory) NewResourcesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) features.ResourcesClient {
	return features.NewResourcesClient(environment, subscriptionID, authorizer)
}

func (cf clientFactory) NewVirtualMachinesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) compute.VirtualMachinesClient {
	return compute.NewVirtualMachinesClient(environment, subscriptionID, authorizer)
}

func newAzureSideFetcher(log *logrus.Entry, resourceGroupName string, subscriptionDoc *api.SubscriptionDocument, env env.Core, spAuthorizer autorest.Authorizer, resourceClientFactory ResourceClientFactory, virtualMachinesClientFactory VirtualMachinesClientFactory) azureSideFetcher {
	return azureSideFetcher{
		log:                          log,
		resourceGroupName:            resourceGroupName,
		subscriptionDoc:              subscriptionDoc,
		spAuthorizer:                 spAuthorizer,
		env:                          env,
		resourceClientFactory:        resourceClientFactory,
		virtualMachinesClientFactory: virtualMachinesClientFactory,
	}
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

func NewAzureFetchClient(log *logrus.Entry, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument, env env.Core) (AzureFetchClient, error) {
	resourceGroupName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	spp := doc.OpenShiftCluster.Properties.ServicePrincipalProfile
	tenantID := subscriptionDoc.Subscription.Properties.TenantID
	options := env.Environment().ClientSecretCredentialOptions()
	tokenCredential, err := azidentity.NewClientSecretCredential(
		tenantID, spp.ClientID, string(spp.ClientSecret), options)
	if err != nil {
		return nil, err
	}
	scopes := []string{env.Environment().ResourceManagerScope}
	spAuthorizer := azidext.NewTokenCredentialAdapter(tokenCredential, scopes)
	cf := clientFactory{}
	azureSideFetcher := newAzureSideFetcher(log, resourceGroupName, subscriptionDoc, env, spAuthorizer, cf, cf)
	return &azureClient{
		log:     log,
		fetcher: &azureSideFetcher,
	}, nil
}
