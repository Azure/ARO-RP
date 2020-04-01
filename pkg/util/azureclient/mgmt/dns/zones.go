package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
)

// ZonesClient is a minimal interface for azure ZonesClient
type ZonesClient interface {
	ZonesClientAddons
}

type zonesClient struct {
	mgmtdns.ZonesClient
}

var _ ZonesClient = &zonesClient{}

// NewZonesClient creates a new ZonesClient
func NewZonesClient(subscriptionID string, authorizer autorest.Authorizer) ZonesClient {
	client := mgmtdns.NewZonesClient(subscriptionID)
	client.Authorizer = authorizer

	return &zonesClient{
		ZonesClient: client,
	}
}
