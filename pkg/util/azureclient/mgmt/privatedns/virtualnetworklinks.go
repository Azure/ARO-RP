package privatedns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	"github.com/Azure/go-autorest/autorest"
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
func NewVirtualNetworkLinksClient(subscriptionID string, authorizer autorest.Authorizer) VirtualNetworkLinksClient {
	client := mgmtprivatedns.NewVirtualNetworkLinksClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualNetworkLinksClient{
		VirtualNetworkLinksClient: client,
	}
}
