package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	gofrsuuid "github.com/gofrs/uuid"

	"github.com/Azure/go-autorest/autorest"

	mgmtredhatopenshift20250725 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2025-07-25/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// OpenShiftClustersClient is a minimal interface for azure OpenshiftClustersClient
type OpenShiftClustersClient interface {
	ListCredentials(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20250725.OpenShiftClusterCredentials, err error)
	Get(ctx context.Context, resourceGroupName string, resourceName string) (result mgmtredhatopenshift20250725.OpenShiftCluster, err error)
	OpenShiftClustersClientAddons
}

type openShiftClustersClient struct {
	mgmtredhatopenshift20250725.OpenShiftClustersClient
}

var _ OpenShiftClustersClient = &openShiftClustersClient{}

// NewOpenShiftClustersClient creates a new OpenShiftClustersClient
func NewOpenShiftClustersClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) OpenShiftClustersClient {
	var client mgmtredhatopenshift20250725.OpenShiftClustersClient
	if env.IsLocalDevelopmentMode() {
		client = mgmtredhatopenshift20250725.NewOpenShiftClustersClientWithBaseURI("https://localhost:8443", gofrsuuid.FromStringOrNil(subscriptionID))
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // #nosec G402  // CodeQL [SM03511] only used in local development
				},
			},
		}
	} else {
		client = mgmtredhatopenshift20250725.NewOpenShiftClustersClientWithBaseURI(environment.ResourceManagerEndpoint, gofrsuuid.FromStringOrNil(subscriptionID))
		client.Authorizer = authorizer
	}
	client.PollingDelay = 10 * time.Second
	client.PollingDuration = 2 * time.Hour

	return &openShiftClustersClient{
		OpenShiftClustersClient: client,
	}
}
