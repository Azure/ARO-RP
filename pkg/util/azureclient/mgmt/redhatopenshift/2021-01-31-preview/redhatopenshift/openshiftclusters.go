package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"

	mgmtredhatopenshift20210131preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// OpenShiftClustersClient is a minimal interface for azure OpenshiftClustersClient
type OpenShiftClustersClient interface {
	ListCredentials(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20210131preview.OpenShiftClusterCredentials, err error)
	Get(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20210131preview.OpenShiftCluster, err error)
	OpenShiftClustersClientAddons
}

type openShiftClustersClient struct {
	mgmtredhatopenshift20210131preview.OpenShiftClustersClient
}

var _ OpenShiftClustersClient = &openShiftClustersClient{}

// NewOpenShiftClustersClient creates a new OpenShiftClustersClient
func NewOpenShiftClustersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) OpenShiftClustersClient {
	var client mgmtredhatopenshift20210131preview.OpenShiftClustersClient
	if env.IsLocalDevelopmentMode() {
		client = mgmtredhatopenshift20210131preview.NewOpenShiftClustersClientWithBaseURI("https://localhost:8443", subscriptionID)
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		client = mgmtredhatopenshift20210131preview.NewOpenShiftClustersClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
		client.Authorizer = authorizer
	}
	client.PollingDelay = 10 * time.Second
	client.PollingDuration = 2 * time.Hour

	return &openShiftClustersClient{
		OpenShiftClustersClient: client,
	}
}
