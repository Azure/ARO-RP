package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

// CapacityReservationGroupsClientAddons contains addons for CapacityReservationGroupsClient
type CapacityReservationGroupsClientAddons interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName, capacityReservationGroupName string, parameters armcompute.CapacityReservationGroup) (armcompute.CapacityReservationGroup, error)
	Delete(ctx context.Context, resourceGroupName, capacityReservationGroupName string) error
}

func (c *capacityReservationGroupsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, capacityReservationGroupName string, parameters armcompute.CapacityReservationGroup) (armcompute.CapacityReservationGroup, error) {
	resp, err := c.CapacityReservationGroupsClient.CreateOrUpdate(ctx, resourceGroupName, capacityReservationGroupName, parameters, nil)
	if err != nil {
		return armcompute.CapacityReservationGroup{}, err
	}
	return resp.CapacityReservationGroup, nil
}

func (c *capacityReservationGroupsClient) Delete(ctx context.Context, resourceGroupName, capacityReservationGroupName string) error {
	_, err := c.CapacityReservationGroupsClient.Delete(ctx, resourceGroupName, capacityReservationGroupName, nil)
	return err
}
