package armcompute

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type CapacityReservationGroupsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, parameters armcompute.CapacityReservationGroup, options *armcompute.CapacityReservationGroupsClientCreateOrUpdateOptions) (armcompute.CapacityReservationGroupsClientCreateOrUpdateResponse, error)
}

type capacityReservationGroupsClient struct {
	*armcompute.CapacityReservationGroupsClient
}

var _ CapacityReservationGroupsClient = &capacityReservationGroupsClient{}

// NewDefaultCapacityReservationGroupsClient creates a new CapacityReservationGroupsClient with default options
func NewDefaultCapacityReservationGroupsClient(environment *azureclient.AROEnvironment, subscriptionId string, credential azcore.TokenCredential) (CapacityReservationGroupsClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}

	return NewCapacityReservationGroupsClient(subscriptionId, credential, options)
}

// NewCapacityReservationGroupsClient creates a new CapacityReservationGroupsClient
func NewCapacityReservationGroupsClient(subscriptionId string, credential azcore.TokenCredential, options *arm.ClientOptions) (CapacityReservationGroupsClient, error) {
	clientFactory, err := armcompute.NewClientFactory(subscriptionId, credential, options)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewCapacityReservationGroupsClient()

	return &capacityReservationGroupsClient{
		CapacityReservationGroupsClient: client,
	}, nil
}
