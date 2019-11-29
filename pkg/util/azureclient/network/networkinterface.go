package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/jim-minter/rp/pkg/util/azureclient"
)

// InterfacesClient is a minimal interface for azure NewInterfacesClient
type InterfacesClient interface {
	Get(ctx context.Context, resourceGroupName string, networkInterfaceName string, expand string) (network.Interface, error)
	Delete(ctx context.Context, resourceGroupName string, networkInterfaceName string) (result network.InterfacesDeleteFuture, err error)
	azureclient.Client
}

type interfacesClient struct {
	network.InterfacesClient
}

var _ InterfacesClient = &interfacesClient{}

// NewInterfacesClient creates a new InterfacesClient
func NewInterfacesClient(subscriptionID string, authorizer autorest.Authorizer) InterfacesClient {
	client := network.NewInterfacesClient(subscriptionID)
	client.Authorizer = authorizer

	return &interfacesClient{
		InterfacesClient: client,
	}
}

func (c *interfacesClient) Client() autorest.Client {
	return c.InterfacesClient.Client
}
