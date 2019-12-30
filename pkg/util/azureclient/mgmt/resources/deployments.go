package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../../vendor/github.com/golang/mock/mockgen -destination=../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/$GOPACKAGE DeploymentsClient,GroupsClient
//go:generate go run ../../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../../util/mocks/azureclient/mgmt/$GOPACKAGE/$GOPACKAGE.go

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

// DeploymentsClient is a minimal interface for azure DeploymentsClient
type DeploymentsClient interface {
	DeploymentsClientAddons
}

type deploymentsClient struct {
	resources.DeploymentsClient
}

var _ DeploymentsClient = &deploymentsClient{}

// NewDeploymentsClient creates a new DeploymentsClient
func NewDeploymentsClient(subscriptionID string, authorizer autorest.Authorizer) DeploymentsClient {
	client := resources.NewDeploymentsClient(subscriptionID)
	client.Authorizer = authorizer
	client.PollingDuration = time.Hour
	client.PollingDelay = 10 * time.Second

	return &deploymentsClient{
		DeploymentsClient: client,
	}
}
