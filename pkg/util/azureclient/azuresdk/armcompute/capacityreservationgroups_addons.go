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

// CapacityReservationGroupsClientAddons is a convenience interface that wraps the SDK CapacityReservationGroupsClient
// with simplified method signatures (no options parameters).
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
	// The SDK's Delete only treats 200/204 as success; all other status codes
	// (including 202 Accepted) are returned as *azcore.ResponseError. When Azure
	// returns 202 the resource is still being deleted asynchronously, so we poll
	// Get until a 404 confirms deletion before returning to the caller.
	// Note: CapacityReservationGroupsClient has no BeginDelete/LRO method in the
	// SDK, so manual polling is the only option.
	_, err := c.CapacityReservationGroupsClient.Delete(ctx, resourceGroupName, capacityReservationGroupName, nil)
	if err != nil {
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
	ticker := time.NewTicker(crgDeletePollInterval)
	defer ticker.Stop()
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
		case <-ticker.C:
		}
	}
}
