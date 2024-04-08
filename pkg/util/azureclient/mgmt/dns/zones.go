package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

// ZonesClient is a minimal interface for azure ZonesClient
type ZonesClient interface {
	Get(ctx context.Context, resourceGroupName string, zoneName string) (result mgmtdns.Zone, err error)
}

type zonesClient struct {
	mgmtdns.ZonesClient
}

var _ ZonesClient = &zonesClient{}

// NewZonesClient creates a new ZonesClient
func NewZonesClient(environment *azureclient.AROEnvironment, subscriptionID string, authorizer autorest.Authorizer) ZonesClient {
	client := mgmtdns.NewZonesClientWithBaseURI(environment.ResourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	client.Sender = azureclient.DecorateSenderWithLogging(client.Sender)

	return &zonesClient{
		ZonesClient: client,
	}
}
