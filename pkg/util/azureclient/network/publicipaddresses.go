package network

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// PublicIPAddressesClient is a minimal interface for azure PublicIPAddressesClient
type PublicIPAddressesClient interface {
	PublicIPAddressesClientAddons
}

type publicIPAddressesClient struct {
	network.PublicIPAddressesClient
}

var _ PublicIPAddressesClient = &publicIPAddressesClient{}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient
func NewPublicIPAddressesClient(subscriptionID string, authorizer autorest.Authorizer) PublicIPAddressesClient {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	client.Authorizer = authorizer

	return &publicIPAddressesClient{
		PublicIPAddressesClient: client,
	}
}
