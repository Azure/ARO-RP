package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/jim-minter/rp/pkg/util/azureclient"
)

// PublicIPAddressesClient is a minimal interface for azure NewPublicIPAddressesClient
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, expand string) (network.PublicIPAddress, error)
	ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx context.Context, resourceGroupName string, scaleSetName string) (network.PublicIPAddressListResultIterator, error)
	Delete(ctx context.Context, resourceGroupName string, publicIPAddressName string) (result network.PublicIPAddressesDeleteFuture, err error)
	azureclient.Client
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

func (c *publicIPAddressesClient) Client() autorest.Client {
	return c.PublicIPAddressesClient.Client
}
