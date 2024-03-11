package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// PrivateZonesClient is a minimal interface for azure PrivateZonesClient
type PrivateZonesClient interface {
	PrivateZonesClientAddons
}

type privateZonesClient struct {
	mgmtprivatedns.PrivateZonesClient
}

var _ PrivateZonesClient = &privateZonesClient{}

// NewPrivateZonesClient creates a new PrivateZonesClient
func NewPrivateZonesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) PrivateZonesClient {
	client := mgmtprivatedns.NewPrivateZonesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &privateZonesClient{
		PrivateZonesClient: client,
	}
}
