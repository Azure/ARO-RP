package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	gofrsuuid "github.com/gofrs/uuid"

	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// OpenShiftClustersClient is a minimal interface for azure OpenshiftClustersClient
type OpenShiftClustersClient interface {
	ListCredentials(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20240812preview.OpenShiftClusterCredentials, err error)
	Get(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20240812preview.OpenShiftCluster, err error)
	OpenShiftClustersClientAddons
}

type openShiftClustersClient struct {
	mgmtredhatopenshift20240812preview.OpenShiftClustersClient
}

var _ OpenShiftClustersClient = &openShiftClustersClient{}

// NewOpenShiftClustersClient creates a new OpenShiftClustersClient
func NewOpenShiftClustersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) OpenShiftClustersClient {
	var client mgmtredhatopenshift20240812preview.OpenShiftClustersClient
	if env.IsLocalDevelopmentMode() {
		client = mgmtredhatopenshift20240812preview.NewOpenShiftClustersClientWithBaseURI("https://localhost:8443", gofrsuuid.FromStringOrNil(subscriptionID))
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // #nosec G402
				},
			},
		}
	} else {
		client = mgmtredhatopenshift20240812preview.NewOpenShiftClustersClientWithBaseURI(environment.ResourceManagerEndpoint, gofrsuuid.FromStringOrNil(subscriptionID))
		client.Authorizer = authorizer
	}
	client.PollingDelay = 10 * time.Second
	client.PollingDuration = 2 * time.Hour

	return &openShiftClustersClient{
		OpenShiftClustersClient: client,
	}
}
