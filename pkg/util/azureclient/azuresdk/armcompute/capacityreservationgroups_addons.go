package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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
	if err != nil {
		// Azure CRG deletion can return 202 Accepted (async). Poll with Get until
		// the resource is gone (404) so callers don't proceed with a stale name.
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusAccepted {
			return c.pollCRGDeleted(ctx, resourceGroupName, capacityReservationGroupName)
		}
		return err
	}
	return nil
}

const crgDeletePollInterval = 5 * time.Second

// pollCRGDeleted polls Get on the CRG until it returns 404 (deleted) or the
// context is cancelled. It is called after a 202 Accepted delete response.
func (c *capacityReservationGroupsClient) pollCRGDeleted(ctx context.Context, resourceGroupName, capacityReservationGroupName string) error {
	for {
		_, err := c.Get(ctx, resourceGroupName, capacityReservationGroupName, nil)
		if err != nil {
			var responseErr *azcore.ResponseError
			if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(crgDeletePollInterval):
		}
	}
}
