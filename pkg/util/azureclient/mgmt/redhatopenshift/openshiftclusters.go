package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../util/mocks/mock_azureclient/mgmt/mock_$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/$GOPACKAGE OpenShiftClustersClient,OperationsClient
//go:generate go run ../../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../../util/mocks/mock_azureclient/mgmt/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/client/services/preview/redhatopenshift/mgmt/2019-12-31-preview/redhatopenshift"
)

// OpenShiftClustersClient is a minimal interface for azure OpenshiftClustersClient
type OpenShiftClustersClient interface {
	ListCredentials(ctx context.Context, resourceGroupName string, resourceName string) (result redhatopenshift.OpenShiftClusterCredentials, err error)
	Get(ctx context.Context, resourceGroupName string, resourceName string) (result redhatopenshift.OpenShiftCluster, err error)
	List(ctx context.Context) (result redhatopenshift.OpenShiftClusterList, err error)
	ListByResourceGroup(ctx context.Context, resourceGroupName string) (result redhatopenshift.OpenShiftClusterList, err error)
}

type openShiftClustersClient struct {
	redhatopenshift.OpenShiftClustersClient
}

var _ OpenShiftClustersClient = &openShiftClustersClient{}

// NewOpenShiftClustersClient creates a new OpenShiftClustersClient
func NewOpenShiftClustersClient(subscriptionID string, authorizer autorest.Authorizer) OpenShiftClustersClient {
	var client redhatopenshift.OpenShiftClustersClient
	if os.Getenv("RP_MODE") == "development" {
		client = redhatopenshift.NewOpenShiftClustersClientWithBaseURI("https://localhost:8443", subscriptionID)
		client.Sender = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		client = redhatopenshift.NewOpenShiftClustersClient(subscriptionID)
		client.Authorizer = authorizer
	}
	client.PollingDuration = time.Minute * 60

	return &openShiftClustersClient{
		OpenShiftClustersClient: client,
	}
}
