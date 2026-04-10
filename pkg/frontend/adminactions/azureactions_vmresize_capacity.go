package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const (
	capacityReservationGroupName = "aro-resize-crg"
	targetReservationNameFmt     = "cr-target-z%s"
)

// CRGResizeSingleVM performs the Azure-side resize of a single VM using a transient
// Capacity Reservation Group to guarantee capacity for the target SKU.
// It is intended to be called from a meta-resize action after the node has been
// cordoned and drained, and before the node is uncordoned.
//
// Flow:
//  1. Create a CRG scoped to the VM's zone.
//  2. Reserve target-SKU capacity in that zone.
//  3. Deallocate the VM.
//  4. Update the VM's SKU and associate it with the CRG in a single call.
//  5. Start the VM (reservation keeps capacity guaranteed during allocation).
//  6. Tear down the CRG (zero reservation, disassociate VM, delete reservation, delete CRG).
//
// The VM is started (step 5) while still associated so the reservation guarantees
// capacity for the allocation. Cleanup happens only after the VM is confirmed running.
// On any failure after CRG creation, a best-effort cleanup is attempted before returning.
func (a *azureActions) CRGResizeSingleVM(ctx context.Context, clusterRG, location, vmName, zone, targetVMSize string) error {
	a.log.Infof("creating capacity reservation group for VM %s (zone %s)", vmName, zone)
	crgID, err := a.CRGCreate(ctx, clusterRG, location, []string{zone})
	if err != nil {
		return err
	}

	// cleanupCRG is called on any failure path after the CRG exists.
	// vmNames should be non-nil only if the VM was successfully associated in step 4.
	cleanupCRG := func(vmNames []string) {
		if cleanErr := a.CRGDelete(ctx, clusterRG, location, targetVMSize, []string{zone}, vmNames); cleanErr != nil {
			a.log.Errorf("CRG cleanup failed for VM %s: %v", vmName, cleanErr)
		}
	}

	a.log.Infof("reserving capacity for VM %s (SKU %s, zone %s)", vmName, targetVMSize, zone)
	if err = a.CRGEnsureReservations(ctx, clusterRG, location, zone, targetVMSize); err != nil {
		cleanupCRG(nil)
		return err
	}

	a.log.Infof("deallocating VM %s before resize", vmName)
	if err = a.armVirtualMachines.DeallocateAndWait(ctx, clusterRG, vmName); err != nil {
		cleanupCRG(nil)
		return fmt.Errorf("deallocating VM %s: %w", vmName, err)
	}

	// Re-read the VM after deallocation to avoid stale-state conflicts.
	vm, err := a.armVirtualMachines.Get(ctx, clusterRG, vmName)
	if err != nil {
		cleanupCRG(nil)
		return fmt.Errorf("reading VM %s before resize: %w", vmName, err)
	}

	// Update the SKU and associate with the CRG in a single call.
	// Association ensures the capacity reservation covers the new SKU during the resize.
	vm.Properties.HardwareProfile.VMSize = (*armcompute.VirtualMachineSizeTypes)(&targetVMSize)
	vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
		CapacityReservationGroup: &armcompute.SubResource{ID: &crgID},
	}

	a.log.Infof("resizing VM %s to %s (with capacity reservation)", vmName, targetVMSize)
	if err = a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, vm); err != nil {
		// The update failed; the VM is unlikely to be associated. Pass nil to skip
		// disassociation in cleanup so it doesn't fail on a non-existent association.
		cleanupCRG(nil)
		return fmt.Errorf("resizing VM %s to %s: %w", vmName, targetVMSize, err)
	}

	// Start the VM while still associated with the CRG so the reservation guarantees
	// capacity for this allocation. Clean up only after the VM is confirmed running.
	a.log.Infof("starting VM %s after resize", vmName)
	if err = a.armVirtualMachines.StartAndWait(ctx, clusterRG, vmName); err != nil {
		cleanupCRG([]string{vmName})
		return fmt.Errorf("starting VM %s after resize: %w", vmName, err)
	}

	// VM is running — release the reservation resources.
	a.log.Infof("cleaning up capacity reservation group after resize of VM %s", vmName)
	if err = a.CRGDelete(ctx, clusterRG, location, targetVMSize, []string{zone}, []string{vmName}); err != nil {
		return fmt.Errorf("cleaning up capacity reservation group: %w", err)
	}
	return nil
}

// CRGCreate creates a Capacity Reservation Group scoped to the given zones.
// Returns the ARM resource ID of the created CRG.
// Azure requires the CRG to declare all zones it will serve at creation time.
func (a *azureActions) CRGCreate(ctx context.Context, clusterRG, location string, zones []string) (string, error) {
	a.log.Infof("creating capacity reservation group %q in zones %v", capacityReservationGroupName, zones)
	crg, err := a.armCapacityReservationGroups.CreateOrUpdate(ctx, clusterRG, capacityReservationGroupName,
		armcompute.CapacityReservationGroup{
			Location: &location,
			Zones:    pointerutils.ToSlicePtr(zones),
		})
	if err != nil {
		if azureerrors.HasAuthorizationFailedError(err) {
			return "", fmt.Errorf(
				"insufficient permissions to manage capacity reservation group in %s "+
					"— requires Microsoft.Compute/capacityReservationGroups/* on resource group %s: %w",
				location, clusterRG, err)
		}
		return "", fmt.Errorf("creating capacity reservation group: %w", err)
	}
	return *crg.ID, nil
}

// CRGEnsureReservations creates a target-SKU capacity reservation for a single zone,
// guaranteeing capacity for the resize destination.
func (a *azureActions) CRGEnsureReservations(ctx context.Context, clusterRG, location, zone, targetSKU string) error {
	crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
	a.log.Infof("creating target-SKU reservation %s (SKU %s) in zone %s", crTarget, targetSKU, zone)
	if err := a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, capacityReservationGroupName, crTarget,
		armcompute.CapacityReservation{
			Location: &location,
			SKU:      &armcompute.SKU{Name: &targetSKU, Capacity: pointerutils.ToPtr(int64(1))},
			Zones:    []*string{pointerutils.ToPtr(zone)},
		}); err != nil {
		if isCapacityError(err) {
			// No automatic fallback is attempted. The caller must choose a different VM family and retry.
			return fmt.Errorf(
				"no capacity available for SKU %s in zone %s — resize aborted, no VMs were modified; "+
					"please retry with a different VM family: %w",
				targetSKU, zone, err)
		}
		return fmt.Errorf("creating target-SKU reservation for zone %s: %w", zone, err)
	}
	return nil
}

// CRGAssociateVM associates a single VM with the named CRG.
// The VM is fetched fresh to avoid stale-state update conflicts.
func (a *azureActions) CRGAssociateVM(ctx context.Context, clusterRG, vmName, crgID string) error {
	vm, err := a.armVirtualMachines.Get(ctx, clusterRG, vmName)
	if err != nil {
		return fmt.Errorf("reading VM %s before association: %w", vmName, err)
	}
	vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
		CapacityReservationGroup: &armcompute.SubResource{ID: &crgID},
	}
	return a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, vm)
}

// CRGDelete deletes the resize CRG and all its capacity reservations.
// Azure requires this specific sequence to avoid constraint violations:
//  1. Set each reservation's capacity to 0 (while VMs are still associated).
//  2. Disassociate all VMs from the CRG (GET + PUT with empty SubResource).
//  3. Delete each reservation.
//  4. Delete the CRG (must be empty).
//
// Zeroing capacity BEFORE disassociating is critical: it removes the allocation
// constraint that causes 409 errors on reservation delete even when VMs are associated.
// vmNames may be nil or empty if no VMs have been associated yet.
// All errors are collected and joined — cleanup continues even if individual steps fail.
func (a *azureActions) CRGDelete(ctx context.Context, clusterRG, location, targetSKU string, zones []string, vmNames []string) error {
	var errs []error

	// Step 1: zero each reservation's capacity BEFORE disassociating VMs.
	// With capacity=0 the reservation holds no allocation, so Azure allows deletion
	// even if its virtualMachinesAssociated list hasn't fully propagated yet.
	for _, zone := range zones {
		crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
		if err := a.setReservationCapacityZero(ctx, location, clusterRG, crTarget, zone, targetSKU); err != nil {
			errs = append(errs, fmt.Errorf("set target reservation %s capacity to 0: %w", crTarget, err))
		}
	}

	// Step 2: disassociate each VM from the CRG.
	// GET the full VM then PUT with an empty SubResource — this is identical to what
	// `az vm update --capacity-reservation-group None` sends and correctly clears the field.
	for _, vmName := range vmNames {
		vm, getErr := a.armVirtualMachines.Get(ctx, clusterRG, vmName)
		if getErr != nil {
			errs = append(errs, fmt.Errorf("read VM %s before disassociation: %w", vmName, getErr))
			continue
		}
		vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
			CapacityReservationGroup: &armcompute.SubResource{},
		}
		if err := a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, vm); err != nil {
			errs = append(errs, fmt.Errorf("disassociate VM %s from CRG: %w", vmName, err))
		}
	}

	// Step 3: delete each reservation (capacity is already 0).
	for _, zone := range zones {
		crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
		if err := a.deleteReservationWithRetry(ctx, clusterRG, crTarget); err != nil {
			errs = append(errs, err)
		}
	}

	// Step 4: delete the CRG last, once all reservations are gone.
	if err := a.deleteCRGWithRetry(ctx, clusterRG); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

const (
	crgRetryInterval = 30 * time.Second
	crgMaxRetries    = 15 // 7.5 minutes maximum
)

// deleteReservationWithRetry deletes a capacity reservation, retrying on 409 "OperationNotAllowed"
// (still referenced by VM) because Azure's reservation bookkeeping lags the VM property update
// by several minutes after a PUT disassociation.
func (a *azureActions) deleteReservationWithRetry(ctx context.Context, clusterRG, crName string) error {
	for attempt := 1; attempt <= crgMaxRetries; attempt++ {
		err := a.armCapacityReservations.DeleteAndWait(ctx, clusterRG, capacityReservationGroupName, crName)
		if err == nil {
			return nil
		}
		if !isReferencedByVMError(err) || attempt == crgMaxRetries {
			return fmt.Errorf("delete target reservation %s: %w", crName, err)
		}
		a.log.Infof("reservation %s still referenced by VM (Azure propagation lag), retrying in %s (attempt %d/%d)",
			crName, crgRetryInterval, attempt, crgMaxRetries)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(crgRetryInterval):
		}
	}
	return nil
}

// deleteCRGWithRetry deletes the capacity reservation group, retrying on 409 "CannotDeleteResource"
// (nested reservations still visible in Azure's resource hierarchy bookkeeping after deletion).
func (a *azureActions) deleteCRGWithRetry(ctx context.Context, clusterRG string) error {
	for attempt := 1; attempt <= crgMaxRetries; attempt++ {
		err := a.armCapacityReservationGroups.Delete(ctx, clusterRG, capacityReservationGroupName)
		if err == nil {
			return nil
		}
		if !isNestedResourcesError(err) || attempt == crgMaxRetries {
			return fmt.Errorf("delete capacity reservation group: %w", err)
		}
		a.log.Infof("CRG still has nested reservations (Azure propagation lag), retrying in %s (attempt %d/%d)",
			crgRetryInterval, attempt, crgMaxRetries)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(crgRetryInterval):
		}
	}
	return nil
}

// setReservationCapacityZero sets a capacity reservation's capacity to 0.
// Azure requires capacity to be 0 before a reservation can be deleted.
func (a *azureActions) setReservationCapacityZero(ctx context.Context, location, clusterRG, crName, zone, skuName string) error {
	return a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, capacityReservationGroupName, crName,
		armcompute.CapacityReservation{
			Location: &location,
			SKU:      &armcompute.SKU{Name: &skuName, Capacity: pointerutils.ToPtr(int64(0))},
			Zones:    []*string{pointerutils.ToPtr(zone)},
		})
}

// isCapacityError returns true when the Azure error indicates insufficient capacity
// rather than a configuration or permission problem.
func isCapacityError(err error) bool {
	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) {
		switch responseError.ErrorCode {
		case "AllocationFailed", "OverconstrainedAllocationRequest", "CapacityReservationCapacityExceeded":
			return true
		}
	}
	return false
}

// isReferencedByVMError returns true when Azure refuses to delete a capacity reservation
// because it is still referenced by a VM (eventual consistency after disassociation PUT).
func isReferencedByVMError(err error) bool {
	var responseError *azcore.ResponseError
	return errors.As(err, &responseError) && responseError.ErrorCode == "OperationNotAllowed"
}

// isNestedResourcesError returns true when Azure refuses to delete a CRG because
// nested reservations are still visible in its resource hierarchy (eventual consistency
// after reservation deletion).
func isNestedResourcesError(err error) bool {
	var responseError *azcore.ResponseError
	return errors.As(err, &responseError) && responseError.ErrorCode == "CannotDeleteResource"
}
