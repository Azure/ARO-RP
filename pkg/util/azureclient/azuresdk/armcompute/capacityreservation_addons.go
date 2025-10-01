package armcompute

import (
	"context"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

type capacityReservationsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, capacityReservationName string, parameters armcompute.CapacityReservation, options *armcompute.CapacityReservationsClientBeginCreateOrUpdateOptions) error
}

func (c *capacityReservationsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, capacityReservationGroupName string, capacityReservationName string, parameters armcompute.CapacityReservation, options *armcompute.CapacityReservationsClientBeginCreateOrUpdateOptions) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, capacityReservationGroupName, capacityReservationName, parameters, options)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
