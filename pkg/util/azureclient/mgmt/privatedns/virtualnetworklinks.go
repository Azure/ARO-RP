package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// VirtualNetworkLinksClient is a minimal interface for azure VirtualNetworkLinksClient
type VirtualNetworkLinksClient interface {
	VirtualNetworkLinksClientAddons
}

type virtualNetworkLinksClient struct {
	mgmtprivatedns.VirtualNetworkLinksClient
}

var _ VirtualNetworkLinksClient = &virtualNetworkLinksClient{}

// NewVirtualNetworkLinksClient creates a new VirtualNetworkLinksClient
func NewVirtualNetworkLinksClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworkLinksClient {
	client := mgmtprivatedns.NewVirtualNetworkLinksClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &virtualNetworkLinksClient{
		VirtualNetworkLinksClient: client,
	}
}
