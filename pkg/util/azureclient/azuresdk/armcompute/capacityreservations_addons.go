package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

// CapacityReservationsClientAddons contains addons for CapacityReservationsClient
type CapacityReservationsClientAddons interface {
	Get(ctx context.Context, resourceGroupName, capacityReservationGroupName, capacityReservationName string, options *armcompute.CapacityReservationsClientGetOptions) (armcompute.CapacityReservationsClientGetResponse, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, capacityReservationGroupName, capacityReservationName string, parameters armcompute.CapacityReservation) error
	DeleteAndWait(ctx context.Context, resourceGroupName, capacityReservationGroupName, capacityReservationName string) error
}

func (c *capacityReservationsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, capacityReservationGroupName, capacityReservationName string, parameters armcompute.CapacityReservation) error {
	poller, err := c.BeginCreateOrUpdate(ctx, resourceGroupName, capacityReservationGroupName, capacityReservationName, parameters, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (c *capacityReservationsClient) DeleteAndWait(ctx context.Context, resourceGroupName, capacityReservationGroupName, capacityReservationName string) error {
	poller, err := c.BeginDelete(ctx, resourceGroupName, capacityReservationGroupName, capacityReservationName, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}
