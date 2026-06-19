package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	targetReservationNameFmt = "cr-target-z%s"
)

// vmIsRunning returns true if the VM's instance view shows PowerState/running.
// Requires the VM to have been fetched with InstanceView expansion (GetWithInstanceView).
func vmIsRunning(vm armcompute.VirtualMachine) bool {
	if vm.Properties == nil || vm.Properties.InstanceView == nil {
		return false
	}
	for _, s := range vm.Properties.InstanceView.Statuses {
		if s.Code != nil && strings.EqualFold(*s.Code, "PowerState/running") {
			return true
		}
	}
	return false
}

// vmIsAtTargetSKUAndRunning returns true if the VM is both at the target SKU and in the
// running power state. Used to detect transient poller errors after a successful resize PUT.
func vmIsAtTargetSKUAndRunning(vm armcompute.VirtualMachine, targetVMSize string) bool {
	if vm.Properties == nil || vm.Properties.HardwareProfile == nil || vm.Properties.HardwareProfile.VMSize == nil {
		return false
	}
	return strings.EqualFold(string(*vm.Properties.HardwareProfile.VMSize), targetVMSize) && vmIsRunning(vm)
}

// crgCreate creates a Capacity Reservation Group scoped to the given zones.
// Returns the ARM resource ID of the created CRG.
// Azure requires the CRG to declare all zones it will serve at creation time.
func (a *azureActions) crgCreate(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error) {
	a.log.Infof("creating capacity reservation group %q in zones %v", crgName, zones)
	crg, err := a.armCapacityReservationGroups.CreateOrUpdate(ctx, clusterRG, crgName,
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
	if crg.ID == nil {
		return "", fmt.Errorf("capacity reservation group %s was created but returned no ID", crgName)
	}
	return *crg.ID, nil
}

// crgEnsureReservations creates a target-SKU capacity reservation for a single zone,
// guaranteeing capacity for the resize destination. capacity must be ≥ 1 and should
// equal the number of VMs to be resized in that zone.
func (a *azureActions) crgEnsureReservations(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error {
	crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
	a.log.Infof("creating target-SKU reservation %s (SKU %s, capacity %d) in zone %s", crTarget, targetSKU, capacity, zone)
	if err := a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, crgName, crTarget,
		armcompute.CapacityReservation{
			Location: &location,
			SKU:      &armcompute.SKU{Name: &targetSKU, Capacity: pointerutils.ToPtr(capacity)},
			Zones:    []*string{pointerutils.ToPtr(zone)},
		}); err != nil {
		if isCapacityError(err) {
			// No automatic fallback is attempted. The caller must choose a different VM family and retry.
			return fmt.Errorf(
				"no capacity available for SKU %s in zone %s — resize aborted, no VMs were modified; "+
					"please retry with a different VM family: %w",
				targetSKU, zone, err)
		}
		if azureerrors.HasAuthorizationFailedError(err) {
			return fmt.Errorf(
				"insufficient permissions to create capacity reservation in %s "+
					"— requires Microsoft.Compute/capacityReservationGroups/capacityReservations/write "+
					"on resource group %s: %w",
				location, clusterRG, err)
		}
		return fmt.Errorf("creating target-SKU reservation for zone %s: %w", zone, err)
	}
	return nil
}

// crgDelete deletes the resize CRG and all its capacity reservations.
// Azure requires this specific sequence to avoid constraint violations:
//  1. Set each reservation's capacity to 0 (while VMs are still associated).
//  2. Disassociate all VMs from the CRG (GET + PUT with capacityReservationGroup: null).
//  3. Delete each reservation.
//  4. Delete the CRG (must be empty).
//
// Zeroing capacity BEFORE disassociating is critical: it removes the allocation
// constraint that causes 409 errors on reservation delete even when VMs are associated.
// vmNames may be nil or empty if no VMs have been associated yet.
// All errors are collected and joined — cleanup continues even if individual steps fail.
func (a *azureActions) crgDelete(ctx context.Context, clusterRG, location, targetSKU string, zones []string, vmNames []string, crgName string) error {
	var errs []error

	// Step 1: zero each reservation's capacity BEFORE disassociating VMs.
	// With capacity=0 the reservation holds no allocation, so Azure allows deletion
	// even if its virtualMachinesAssociated list hasn't fully propagated yet.
	// Each step derives from ctx so caller cancellation is still honoured, but caps
	// the step at its own maximum so one slow ARM call cannot starve later steps.
	for _, zone := range zones {
		crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
		stepCtx, stepCancel := context.WithTimeout(ctx, crgCapacityZeroTimeout)
		err := a.setReservationCapacityZero(stepCtx, location, clusterRG, crTarget, zone, targetSKU, crgName)
		stepCancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("set target reservation %s capacity to 0: %w", crTarget, err))
		}
	}

	// Step 2: disassociate each VM from the CRG.
	// Mirror what `az vm update --capacity-reservation-group None` does: GET the full VM,
	// set capacityReservation.capacityReservationGroup.id = null (azcore.NullValue), then PUT.
	// ARM only clears the association when receiving a full PUT where the id field inside
	// the capacityReservationGroup sub-resource is explicitly null — sending
	// {"capacityReservation": null} or {"capacityReservationGroup": null} both return 200 OK
	// but do NOT clear the reservation's virtualMachinesAssociated list.
	for _, vmName := range vmNames {
		stepCtx, stepCancel := context.WithTimeout(ctx, crgDisassociateVMTimeout)
		vm, getErr := a.armVirtualMachines.Get(stepCtx, clusterRG, vmName)
		if getErr != nil {
			stepCancel()
			errs = append(errs, fmt.Errorf("read VM %s before disassociation: %w", vmName, getErr))
			continue
		}
		if vm.Properties == nil {
			stepCancel()
			errs = append(errs, fmt.Errorf("VM %s has no properties", vmName))
			continue
		}
		if vm.Properties.CapacityReservation == nil || vm.Properties.CapacityReservation.CapacityReservationGroup == nil || vm.Properties.CapacityReservation.CapacityReservationGroup.ID == nil {
			a.log.Infof("VM %s has no capacity reservation group association; skipping disassociation", vmName)
			stepCancel()
			continue
		}
		vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
			CapacityReservationGroup: &armcompute.SubResource{
				ID: azcore.NullValue[*string](),
			},
		}
		a.log.Infof("disassociating VM %s from CRG %s", vmName, crgName)
		if err := a.armVirtualMachines.CreateOrUpdateAndWait(stepCtx, clusterRG, vmName, vm); err != nil {
			errs = append(errs, fmt.Errorf("disassociate VM %s from CRG: %w", vmName, err))
			a.log.Errorf("failed to disassociate VM %s from CRG %s: %v", vmName, crgName, err)
		} else {
			a.log.Infof("successfully disassociated VM %s from CRG %s", vmName, crgName)
		}
		stepCancel()
	}

	// Step 3: delete each reservation (capacity is already 0).
	for _, zone := range zones {
		crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
		stepCtx, stepCancel := context.WithTimeout(ctx, crgDeleteReservationTimeout)
		err := a.deleteReservationWithRetry(stepCtx, clusterRG, crTarget, crgName)
		stepCancel()
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Step 4: delete the CRG last, once all reservations are gone.
	stepCtx, stepCancel := context.WithTimeout(ctx, crgDeleteGroupTimeout)
	defer stepCancel()
	if err := a.deleteCRGWithRetry(stepCtx, clusterRG, crgName); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

const (
	crgMaxRetries = 15 // 7.5 minutes maximum
	// Per-step timeouts for crgDelete steps:
	// Steps 1+2 (capacity zero + VM disassociation) are single ARM PUTs; 5 min each is generous.
	crgCapacityZeroTimeout   = 5 * time.Minute
	crgDisassociateVMTimeout = 5 * time.Minute
	// vmStartTimeout is the budget for restarting a VM during error-path recovery.
	vmStartTimeout = 10 * time.Minute
	// vmProbeTimeout is the budget for the post-failure GET used to check VM state.
	vmProbeTimeout = 30 * time.Second
	// crgOrphanProbeRetries governs the short retry window used by deleteOrphanedCRG to
	// handle the eventual-consistency gap after a create-timeout race.
	crgOrphanProbeRetries = 6
)

// Retry-loop intervals and derived timeouts are vars so tests can override them without
// sleeping for real. The values are intentionally generous for production use.
var (
	// crgRetryInterval is the wait between reservation/CRG delete retry attempts.
	crgRetryInterval = 30 * time.Second
	// Steps 3+4 are retry loops; each loop can run up to crgMaxRetries*crgRetryInterval = 7.5 min.
	crgDeleteReservationTimeout = time.Duration(crgMaxRetries)*crgRetryInterval + time.Minute
	crgDeleteGroupTimeout       = time.Duration(crgMaxRetries)*crgRetryInterval + time.Minute
	// crgCleanupTimeout is an overall budget for the entire crgDelete call.
	// It covers all four steps sequentially with a 2-minute buffer.
	crgCleanupTimeout = crgCapacityZeroTimeout + crgDisassociateVMTimeout + crgDeleteReservationTimeout + crgDeleteGroupTimeout + 2*time.Minute
	// crgOrphanProbeInterval is the wait between deleteOrphanedCRG retry attempts.
	crgOrphanProbeInterval = 5 * time.Second
)

// retryOnAzureEventualConsistency calls op up to maxRetries times, waiting interval
// between attempts for as long as shouldRetry returns true for the error op returned.
// It returns nil as soon as op succeeds, the error immediately once shouldRetry is
// false, the last error once attempts are exhausted, or ctx.Err() if the context is
// cancelled while waiting. The CRG cleanup steps all retry on Azure eventual-consistency
// lag; this keeps the loop, timer, and cancellation handling in one place rather than
// re-deriving it per caller.
func (a *azureActions) retryOnAzureEventualConsistency(ctx context.Context, maxRetries int, interval time.Duration, op func() error, shouldRetry func(error) bool) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = op()
		if lastErr == nil || !shouldRetry(lastErr) || attempt == maxRetries {
			return lastErr
		}
		a.log.Infof("Azure eventual-consistency error, retrying in %s (attempt %d/%d): %v", interval, attempt, maxRetries, lastErr)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

// deleteReservationWithRetry deletes a capacity reservation, retrying on 409 "OperationNotAllowed"
// (still referenced by VM) because Azure's reservation bookkeeping lags the VM property update
// by several minutes after a PUT disassociation.
func (a *azureActions) deleteReservationWithRetry(ctx context.Context, clusterRG, crName, crgName string) error {
	op := func() error {
		err := a.armCapacityReservations.DeleteAndWait(ctx, clusterRG, crgName, crName)
		if err == nil || azureerrors.IsStatusNotFoundError(err) {
			return nil
		}
		return err
	}
	if err := a.retryOnAzureEventualConsistency(ctx, crgMaxRetries, crgRetryInterval, op, isReferencedByVMError); err != nil {
		return fmt.Errorf("delete target reservation %s: %w", crName, err)
	}
	return nil
}

// deleteCRGWithRetry deletes the capacity reservation group, retrying on 409 "CannotDeleteResource"
// (nested reservations still visible in Azure's resource hierarchy bookkeeping after deletion).
func (a *azureActions) deleteCRGWithRetry(ctx context.Context, clusterRG, crgName string) error {
	op := func() error {
		err := a.armCapacityReservationGroups.Delete(ctx, clusterRG, crgName)
		if err == nil || azureerrors.IsStatusNotFoundError(err) {
			return nil
		}
		return err
	}
	if err := a.retryOnAzureEventualConsistency(ctx, crgMaxRetries, crgRetryInterval, op, isNestedResourcesError); err != nil {
		return fmt.Errorf("delete capacity reservation group: %w", err)
	}
	return nil
}

// deleteOrphanedCRG deletes a CRG that may or may not yet exist, retrying on 404
// to handle the eventual-consistency window after a create-timeout race: Azure may
// have persisted the CRG but the delete arrives before it becomes visible.
// After crgOrphanProbeRetries attempts a persistent 404 is accepted as confirmation
// the CRG was never created.
// For non-404 errors it falls back to deleteCRGWithRetry (handles nested-resources 409 etc.).
func (a *azureActions) deleteOrphanedCRG(ctx context.Context, clusterRG, crgName string) error {
	op := func() error { return a.armCapacityReservationGroups.Delete(ctx, clusterRG, crgName) }
	err := a.retryOnAzureEventualConsistency(ctx, crgOrphanProbeRetries, crgOrphanProbeInterval, op, azureerrors.IsStatusNotFoundError)
	switch {
	case err == nil:
		return nil
	case azureerrors.IsStatusNotFoundError(err):
		return nil // persistent 404: CRG was never created
	default:
		// non-404 error: fall back to full retry handling (nested-resources 409 etc.).
		return a.deleteCRGWithRetry(ctx, clusterRG, crgName)
	}
}

// setReservationCapacityZero sets a capacity reservation's capacity to 0.
// Azure requires capacity to be 0 before a reservation can be deleted.
func (a *azureActions) setReservationCapacityZero(ctx context.Context, location, clusterRG, crName, zone, skuName, crgName string) error {
	return a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, crgName, crName,
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
// Requires 409 Conflict + "OperationNotAllowed" + a VM-reference substring to avoid
// retrying unrelated OperationNotAllowed errors (e.g. subscription policy restrictions)
// for the full 7.5-minute budget.
// NOTE: Azure error messages use either "virtual machine" (for standalone VMs) or
// "vmss" (for VMSS members) depending on the resource type. Both substrings are
// checked case-insensitively here.
func isReferencedByVMError(err error) bool {
	var responseError *azcore.ResponseError
	if !errors.As(err, &responseError) ||
		responseError.StatusCode != http.StatusConflict ||
		responseError.ErrorCode != "OperationNotAllowed" {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "virtual machine") || strings.Contains(errStr, "vmss")
}

// isNestedResourcesError returns true when Azure refuses to delete a CRG because
// nested reservations are still visible in its resource hierarchy (eventual consistency
// after reservation deletion).
// Requires 409 Conflict + "CannotDeleteResource" + a "nested resources" substring to avoid
// retrying unrelated CannotDeleteResource errors (e.g. resource locks) for the full
// 7.5-minute budget.
// NOTE: The "nested resources" substring check relies on Azure's English error message text.
// Azure could change the phrasing, which would cause this guard to miss the retry window.
// The substring check is still valuable: without it, any CannotDeleteResource 409 (including
// resource locks) would be retried for the full budget unnecessarily.
func isNestedResourcesError(err error) bool {
	var responseError *azcore.ResponseError
	return errors.As(err, &responseError) &&
		responseError.StatusCode == http.StatusConflict &&
		responseError.ErrorCode == "CannotDeleteResource" &&
		strings.Contains(strings.ToLower(err.Error()), "nested resources")
}

// CRGSetupForResize creates a single shared Capacity Reservation Group for a set of VMs,
// reserving target-SKU capacity in each VM's detected availability zone. All VMs must be
// zonal; non-zonal clusters should use the plain VMResize path instead.
//
// When multiple VMs share an availability zone, the reservation capacity for that zone is
// set equal to the number of VMs in it, ensuring all of them can associate simultaneously.
//
// Returns the CRG ID, a unique per-operation CRG name, and the deduplicated list of zones
// so the caller can pass them to CRGTeardown after all per-VM resizes complete.
// On any failure after CRG creation, a best-effort cleanup is attempted before returning.
func (a *azureActions) CRGSetupForResize(ctx context.Context, vmNames []string, targetSKU string) (string, string, []string, error) {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	location := a.oc.Location

	// Detect the availability zone for each VM. All must be zonal.
	// Count per-zone so we can reserve the right capacity for clusters where
	// multiple masters land in the same zone.
	zoneCount := make(map[string]int, len(vmNames))
	for _, vmName := range vmNames {
		vm, err := a.armVirtualMachines.Get(ctx, clusterRG, vmName)
		if err != nil {
			return "", "", nil, fmt.Errorf("reading VM %s: %w", vmName, err)
		}
		if len(vm.Zones) == 0 || vm.Zones[0] == nil {
			return "", "", nil, fmt.Errorf("VM %s has no availability zone; capacity reservation requires zonal VMs", vmName)
		}
		zoneCount[*vm.Zones[0]]++
	}

	// Build a sorted deduplicated zone list for deterministic CRG creation.
	uniqueZones := make([]string, 0, len(zoneCount))
	for z := range zoneCount {
		uniqueZones = append(uniqueZones, z)
	}
	sort.Strings(uniqueZones)

	// Use a per-operation unique name to prevent collisions between concurrent resizes
	// or with lingering CRGs from prior failed runs.
	crgName := fmt.Sprintf("aro-resize-crg-cp-%s", uuid.New().String())

	a.log.Infof("creating shared capacity reservation group %s for zones %v (SKU %s)", crgName, uniqueZones, targetSKU)
	crgID, err := a.crgCreate(ctx, clusterRG, location, uniqueZones, crgName)
	if err != nil {
		cleanCtx, cleanCancel := context.WithTimeout(context.WithoutCancel(ctx), crgCleanupTimeout)
		if cleanErr := a.deleteOrphanedCRG(cleanCtx, clusterRG, crgName); cleanErr != nil {
			cleanCancel()
			a.log.Errorf("best-effort cleanup of potentially orphaned CRG %s after failed create: %v", crgName, cleanErr)
			return "", "", nil, errors.Join(err, cleanErr)
		}
		cleanCancel()
		return "", "", nil, err
	}

	for _, zone := range uniqueZones {
		count := int64(zoneCount[zone])
		if err := a.crgEnsureReservations(ctx, clusterRG, location, zone, targetSKU, crgName, count); err != nil {
			cleanCtx, cleanCancel := context.WithTimeout(context.WithoutCancel(ctx), crgCleanupTimeout)
			if cleanErr := a.crgDelete(cleanCtx, clusterRG, location, targetSKU, uniqueZones, nil, crgName); cleanErr != nil {
				cleanCancel()
				a.log.Errorf("CRG cleanup failed after reservation error for zone %s: %v", zone, cleanErr)
				return "", "", nil, errors.Join(err, cleanErr)
			}
			cleanCancel()
			return "", "", nil, err
		}
	}

	return crgID, crgName, uniqueZones, nil
}

// VMResizeWithCRG performs the per-VM resize steps against an already-created shared CRG:
// deallocate → re-read → resize + associate → start. CRG lifecycle is the caller's
// responsibility (see CRGSetupForResize / CRGTeardown).
//
// On any failure, a best-effort restart is attempted so the VM is not left deallocated.
// Transient poller errors are detected by re-reading the VM state after resize/start failures;
// if the VM is confirmed running at the target SKU the error is swallowed and success returned.
func (a *azureActions) VMResizeWithCRG(ctx context.Context, vmName, crgID, targetVMSize string) error {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	// bestEffortRestart attempts to start the VM so it is not left deallocated on failure.
	// CRG management is not done here — the caller holds the teardown responsibility.
	bestEffortRestart := func(reason string) {
		restartCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), vmStartTimeout)
		defer cancel()
		a.log.Infof("attempting to restart VM %s after %s (best-effort)", vmName, reason)
		if err := a.armVirtualMachines.StartAndWait(restartCtx, clusterRG, vmName); err != nil {
			a.log.Errorf("best-effort restart of VM %s failed after %s: %v", vmName, reason, err)
		}
	}

	a.log.Infof("deallocating VM %s before resize", vmName)
	if err := a.armVirtualMachines.DeallocateAndWait(ctx, clusterRG, vmName); err != nil {
		bestEffortRestart("failed deallocation")
		return fmt.Errorf("deallocating VM %s: %w", vmName, err)
	}

	// Re-read the VM after deallocation to avoid stale-state conflicts on the resize PUT.
	vm, err := a.armVirtualMachines.Get(ctx, clusterRG, vmName)
	if err != nil {
		bestEffortRestart("failed VM re-read after deallocation")
		return fmt.Errorf("reading VM %s before resize: %w", vmName, err)
	}

	if vm.Properties == nil || vm.Properties.HardwareProfile == nil {
		bestEffortRestart("missing hardware profile")
		return fmt.Errorf("VM %s has no hardware profile", vmName)
	}

	// Update SKU and associate with the shared CRG in a single ARM PUT.
	size := armcompute.VirtualMachineSizeTypes(targetVMSize)
	vm.Properties.HardwareProfile.VMSize = &size
	vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
		CapacityReservationGroup: &armcompute.SubResource{ID: &crgID},
	}

	a.log.Infof("resizing VM %s to %s (with CRG association)", vmName, targetVMSize)
	if err = a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, vm); err != nil {
		bestEffortRestart("failed resize")
		// The PUT may have partially applied (e.g. timeout before response). Re-read the VM
		// with InstanceView to check both SKU and power state. If the VM is already running
		// at the target SKU, treat as success (bestEffortRestart ensured it's started).
		probeCtx, probeCancel := context.WithTimeout(context.WithoutCancel(ctx), vmProbeTimeout)
		freshVM, probeErr := a.armVirtualMachines.GetWithInstanceView(probeCtx, clusterRG, vmName)
		probeCancel()
		if probeErr == nil && vmIsAtTargetSKUAndRunning(freshVM, targetVMSize) {
			a.log.Warnf("VM %s resize returned transient error but VM is at target SKU %s and running: %v", vmName, targetVMSize, err)
			return nil
		}
		return fmt.Errorf("resizing VM %s to %s: %w", vmName, targetVMSize, err)
	}

	a.log.Infof("starting VM %s after resize", vmName)
	if err = a.armVirtualMachines.StartAndWait(ctx, clusterRG, vmName); err != nil {
		// StartAndWait polls an async operation; error may be transient even if Azure started the VM.
		// Re-read with InstanceView to verify power state before surfacing the error.
		probeCtx, probeCancel := context.WithTimeout(context.WithoutCancel(ctx), vmProbeTimeout)
		freshVM, probeErr := a.armVirtualMachines.GetWithInstanceView(probeCtx, clusterRG, vmName)
		probeCancel()
		if probeErr == nil && vmIsRunning(freshVM) {
			a.log.Warnf("VM %s start returned transient error but VM is running (resize complete): %v", vmName, err)
			return nil
		}
		return fmt.Errorf("starting VM %s after resize: %w", vmName, err)
	}

	return nil
}

// CRGTeardown tears down a shared CRG created by CRGSetupForResize after all per-VM
// resizes have completed (or failed). It derives clusterRG and location from the cluster
// document, matching the convention used by VMResize and other azureActions wrappers.
// A fresh timeout context is derived from ctx so cleanup runs even if ctx is cancelled
// while preserving any tracing values from the original request context.
func (a *azureActions) CRGTeardown(ctx context.Context, targetSKU string, zones, vmNames []string, crgName string) error {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	cleanCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), crgCleanupTimeout)
	defer cancel()
	return a.crgDelete(cleanCtx, clusterRG, a.oc.Location, targetSKU, zones, vmNames, crgName)
}
