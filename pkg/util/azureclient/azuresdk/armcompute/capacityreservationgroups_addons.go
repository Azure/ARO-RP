package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

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
	// The SDK's Delete only treats 200/204 as success; any other status code
	// (including 202 Accepted) is returned as *azcore.ResponseError. We defensively
	// handle 202 in case the service returns it for this resource type, since
	// CapacityReservationGroupsClient has no BeginDelete/LRO method in the SDK to
	// poll it for us: on 202 we poll Get until a 404 confirms deletion before
	// returning to the caller.
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

var (
	crgDeletePollInterval = 5 * time.Second
	// crgDeleteTimeout bounds pollCRGDeleted independently of the caller's context,
	// so a caller-supplied context with no deadline cannot leave this polling
	// indefinitely. Mirrors the same safety-net pattern used by
	// azcertificates.WaitForCertificateOperation.
	crgDeleteTimeout = 15 * time.Minute
)

// pollCRGDeleted polls Get on the CRG until it returns 404 (deleted), the
// context is cancelled, or crgDeleteTimeout elapses. It is called after a
// 202 Accepted delete response.
func (c *capacityReservationGroupsClient) pollCRGDeleted(ctx context.Context, resourceGroupName, capacityReservationGroupName string) error {
	ctx, cancel := context.WithTimeout(ctx, crgDeleteTimeout)
	defer cancel()

	return wait.PollUntilContextCancel(ctx, crgDeletePollInterval, true, func(ctx context.Context) (bool, error) {
		_, err := c.Get(ctx, resourceGroupName, capacityReservationGroupName, nil)
		if err != nil {
			var responseErr *azcore.ResponseError
			if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}
