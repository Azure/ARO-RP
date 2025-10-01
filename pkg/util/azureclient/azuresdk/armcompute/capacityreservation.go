package armcompute

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type CapacityReservationsClient interface {
	capacityReservationsClientAddons
}

type capacityReservationsClient struct {
	*armcompute.CapacityReservationsClient
}

var _ CapacityReservationsClient = &capacityReservationsClient{}

// NewDefaultCapacityReservationsClient creates a new CapacityReservationsClient with default options
func NewDefaultCapacityReservationsClient(environment *azureclient.AROEnvironment, subscriptionId string, credential azcore.TokenCredential) (CapacityReservationsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}

	return NewCapacityReservationsClient(subscriptionId, credential, options)
}

// NewCapacityReservationsClient creates a new CapacityReservationsClient
func NewCapacityReservationsClient(subscriptionId string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionId, credential, options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewCapacityReservationsClient()

	return &capacityReservationsClient{
		CapacityReservationsClient: client,
	}, nil
}
