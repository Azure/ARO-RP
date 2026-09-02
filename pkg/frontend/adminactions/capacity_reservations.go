package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const targetReservationNameFmt = "cr-target-z%s"

func (a *azureActions) CreateCRG(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error) {
	var crg armcompute.CapacityReservationGroup
	err := arm.Retryable(ctx, func() error {
		var createErr error
		crg, createErr = a.capacityReservationGroups.CreateOrUpdate(ctx, clusterRG, crgName,
			armcompute.CapacityReservationGroup{
				Location: &location,
				Zones:    pointerutils.ToSlicePtr(zones),
			})
		return createErr
	}, a.log, fmt.Sprintf("creating capacity reservation group %s", crgName))
	if err != nil {
		return "", err
	}
	if crg.ID == nil {
		return "", fmt.Errorf("capacity reservation group %s created but returned no ID", crgName)
	}
	return *crg.ID, nil
}

func (a *azureActions) CreateCapacityReservation(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error {
	crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
	return arm.Retryable(ctx, func() error {
		return a.capacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, crgName, crTarget,
			armcompute.CapacityReservation{
				Location: &location,
				SKU:      &armcompute.SKU{Name: &targetSKU, Capacity: pointerutils.ToPtr(capacity)},
				Zones:    []*string{pointerutils.ToPtr(zone)},
			})
	}, a.log, fmt.Sprintf("create target-SKU reservation in zone %s", zone))
}

func (a *azureActions) DeleteCRG(ctx context.Context, clusterRG, crgName string) error {
	return arm.RetryableDelete(ctx, func() error {
		return a.capacityReservationGroups.Delete(ctx, clusterRG, crgName)
	}, a.log, fmt.Sprintf("deleting capacity reservation group %s", crgName))
}

func (a *azureActions) DeleteCapacityReservation(ctx context.Context, clusterRG, crgName, zone string) error {
	crName := fmt.Sprintf(targetReservationNameFmt, zone)
	return arm.RetryableDelete(ctx, func() error {
		return a.capacityReservations.DeleteAndWait(ctx, clusterRG, crgName, crName)
	}, a.log, fmt.Sprintf("deleting capacity reservation %s from CRG %s", crName, crgName))
}
