package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// Shared with the CRG teardown path, which deletes reservations named by this format.
const targetReservationNameFmt = "cr-target-z%s"

const vmProbeTimeout = 30 * time.Second

const vmStartTimeout = 10 * time.Minute

// Requires the VM to have been fetched with InstanceView expansion (GetWithInstanceView).
//
// This duplicates the intent (not the shape) of validateVMPowerState in
// pkg/frontend/admin_openshiftcluster_resize_validation_helpers.go: that helper enforces a
// strict pre-flight gate (exactly 2 statuses, both healthy) on the old autorest mgmtcompute
// []string status shape, while this is a loose post-operation probe on the Track2 armcompute
// InstanceView. TODO: merge the two once azureactions.go moves off the old autorest
// mgmtcompute SDK onto the Track2 armcompute client.
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

// vmIsAtTargetSKUAndRunning detects transient poller errors after a successful resize PUT.
func vmIsAtTargetSKUAndRunning(vm armcompute.VirtualMachine, targetVMSize string) bool {
	if vm.Properties == nil || vm.Properties.HardwareProfile == nil || vm.Properties.HardwareProfile.VMSize == nil {
		return false
	}
	return strings.EqualFold(string(*vm.Properties.HardwareProfile.VMSize), targetVMSize) && vmIsRunning(vm)
}

// Azure requires the CRG to declare all zones it will serve at creation time.
func (a *azureActions) crgCreate(ctx context.Context, clusterRG, location string, zones []string, crgName string) (string, error) {
	a.log.Infof("creating capacity reservation group %q in zones %v", crgName, zones)
	var crg armcompute.CapacityReservationGroup
	err := arm.Retryable(ctx, func() error {
		var createErr error
		crg, createErr = a.armCapacityReservationGroups.CreateOrUpdate(ctx, clusterRG, crgName,
			armcompute.CapacityReservationGroup{
				Location: &location,
				Zones:    pointerutils.ToSlicePtr(zones),
			})
		return createErr
	}, a.log, fmt.Sprintf("create capacity reservation group %s", crgName))
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

// capacity must be ≥ 1 and should equal the number of VMs to be resized in that zone.
func (a *azureActions) crgEnsureReservations(ctx context.Context, clusterRG, location, zone, targetSKU, crgName string, capacity int64) error {
	crTarget := fmt.Sprintf(targetReservationNameFmt, zone)
	a.log.Infof("creating target-SKU reservation %s (SKU %s, capacity %d) in zone %s", crTarget, targetSKU, capacity, zone)
	err := arm.Retryable(ctx, func() error {
		return a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, crgName, crTarget,
			armcompute.CapacityReservation{
				Location: &location,
				SKU:      &armcompute.SKU{Name: &targetSKU, Capacity: pointerutils.ToPtr(capacity)},
				Zones:    []*string{pointerutils.ToPtr(zone)},
			})
	}, a.log, fmt.Sprintf("create target-SKU reservation %s in zone %s", crTarget, zone))
	if err != nil {
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

// VMs already at targetSKU are excluded from capacity sizing and from the returned
// resizeVMNames. If every VM is already at the target SKU, no CRG is created and all
// returned values are empty. On partial failure, no cleanup is attempted — CRGTeardown
// handles removal of the partial CRG and any reservations already created.
func (a *azureActions) CRGSetupForResize(ctx context.Context, vmNames []string, targetSKU string) (crgID, crgName string, zones, resizeVMNames []string, err error) {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	location := a.oc.Location

	zoneCount := make(map[string]int, len(vmNames))
	resizeVMNames = make([]string, 0, len(vmNames))
	for _, vmName := range vmNames {
		vm, err := a.armVirtualMachines.GetDefault(ctx, clusterRG, vmName)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("reading VM %s: %w", vmName, err)
		}
		if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil &&
			strings.EqualFold(string(*vm.Properties.HardwareProfile.VMSize), targetSKU) {
			a.log.Infof("VM %s is already at target SKU %s; excluding from capacity reservation sizing", vmName, targetSKU)
			continue
		}
		if len(vm.Zones) == 0 || vm.Zones[0] == nil {
			return "", "", nil, nil, fmt.Errorf("VM %s has no availability zone; capacity reservation requires zonal VMs", vmName)
		}
		zoneCount[*vm.Zones[0]]++
		resizeVMNames = append(resizeVMNames, vmName)
	}

	if len(zoneCount) == 0 {
		if len(vmNames) == 0 {
			a.log.Infof("no VMs given; no capacity reservation group needed")
		} else {
			a.log.Infof("all VMs already at target SKU %s; no capacity reservation group needed", targetSKU)
		}
		return "", "", nil, nil, nil
	}

	// Sorted for deterministic CRG creation.
	uniqueZones := make([]string, 0, len(zoneCount))
	for z := range zoneCount {
		uniqueZones = append(uniqueZones, z)
	}
	sort.Strings(uniqueZones)

	// Use a per-operation unique name to prevent collisions between concurrent resizes
	// or with lingering CRGs from prior failed runs.
	crgName = fmt.Sprintf("aro-resize-crg-cp-%s", uuid.New().String())

	a.log.Infof("creating shared capacity reservation group %s for zones %v (SKU %s)", crgName, uniqueZones, targetSKU)
	crgID, err = a.crgCreate(ctx, clusterRG, location, uniqueZones, crgName)
	if err != nil {
		return "", "", nil, nil, err
	}

	for _, zone := range uniqueZones {
		count := int64(zoneCount[zone])
		if err := a.crgEnsureReservations(ctx, clusterRG, location, zone, targetSKU, crgName, count); err != nil {
			return "", "", nil, nil, err
		}
	}

	return crgID, crgName, uniqueZones, resizeVMNames, nil
}

// CRG lifecycle is the caller's responsibility (see CRGSetupForResize / CRGTeardown).
// On any failure, a best-effort restart is attempted so the VM is not left deallocated.
// Transient poller errors are resolved by re-reading VM state; if the VM is confirmed
// running at the target SKU, the error is swallowed and success is returned.
func (a *azureActions) VMResizeWithCRG(ctx context.Context, vmName, crgID, targetVMSize string) error {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	// CRG management is not done here — the caller holds the teardown responsibility.
	bestEffortRestart := func(reason string) {
		restartCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), vmStartTimeout)
		defer cancel()
		a.log.Infof("attempting to restart VM %s after %s (best-effort)", vmName, reason)
		if err := arm.Retryable(restartCtx, func() error {
			return a.armVirtualMachines.StartAndWait(restartCtx, clusterRG, vmName)
		}, a.log, fmt.Sprintf("best-effort restart of VM %s", vmName)); err != nil {
			a.log.Errorf("best-effort restart of VM %s failed after %s: %v", vmName, reason, err)
		}
	}

	a.log.Infof("deallocating VM %s before resize", vmName)
	if err := arm.Retryable(ctx, func() error {
		return a.armVirtualMachines.DeallocateAndWait(ctx, clusterRG, vmName)
	}, a.log, fmt.Sprintf("deallocate VM %s", vmName)); err != nil {
		bestEffortRestart("failed deallocation")
		return fmt.Errorf("deallocating VM %s: %w", vmName, err)
	}

	// Re-read the VM after deallocation to avoid stale-state conflicts on the resize PUT.
	vm, err := a.armVirtualMachines.GetDefault(ctx, clusterRG, vmName)
	if err != nil {
		bestEffortRestart("failed VM re-read after deallocation")
		return fmt.Errorf("reading VM %s before resize: %w", vmName, err)
	}

	if vm.Properties == nil || vm.Properties.HardwareProfile == nil {
		bestEffortRestart("missing hardware profile")
		return fmt.Errorf("VM %s has no hardware profile", vmName)
	}

	if existing := vm.Properties.CapacityReservation; existing != nil && existing.CapacityReservationGroup != nil && existing.CapacityReservationGroup.ID != nil {
		a.log.Warnf("VM %s is already associated with capacity reservation group %s; overwriting with %s", vmName, *existing.CapacityReservationGroup.ID, crgID)
	}

	size := armcompute.VirtualMachineSizeTypes(targetVMSize)
	vm.Properties.HardwareProfile.VMSize = &size
	vm.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
		CapacityReservationGroup: &armcompute.SubResource{ID: &crgID},
	}

	a.log.Infof("resizing VM %s to %s (with CRG association)", vmName, targetVMSize)
	if err = arm.Retryable(ctx, func() error {
		return a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, vm)
	}, a.log, fmt.Sprintf("resize VM %s to %s", vmName, targetVMSize)); err != nil {
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
	if err = arm.Retryable(ctx, func() error {
		return a.armVirtualMachines.StartAndWait(ctx, clusterRG, vmName)
	}, a.log, fmt.Sprintf("start VM %s after resize", vmName)); err != nil {
		// StartAndWait polls an async operation; error may be transient even if Azure started the VM.
		// Re-read with InstanceView to verify power state before surfacing the error.
		probeCtx, probeCancel := context.WithTimeout(context.WithoutCancel(ctx), vmProbeTimeout)
		freshVM, probeErr := a.armVirtualMachines.GetWithInstanceView(probeCtx, clusterRG, vmName)
		probeCancel()
		if probeErr == nil && vmIsRunning(freshVM) {
			a.log.Warnf("VM %s start returned transient error but VM is running (resize complete): %v", vmName, err)
			return nil
		}
		bestEffortRestart("failed start")
		return fmt.Errorf("starting VM %s after resize: %w", vmName, err)
	}

	return nil
}
