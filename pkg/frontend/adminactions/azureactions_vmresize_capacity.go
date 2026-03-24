package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

const (
	capacityReservationGroupName = "aro-resize-crg"
	currentReservationNameFmt    = "cr-current-z%s"
	targetReservationNameFmt     = "cr-target-z%s"
)

// VMResizeWithCapacityReservation resizes all master VMs to the target SKU using
// Azure Capacity Reservation Groups to guarantee capacity in each availability zone.
//
// Flow:
//  1. List master VMs and record their zones.
//  2. Create a Capacity Reservation Group (CRG).
//  3. Create current-SKU and target-SKU reservations per zone before touching any VM.
//     If target capacity is unavailable the rollback is simple: delete reservations + CRG.
//  4. Associate all master VMs with the CRG.
//  5. Resize each VM one at a time (deallocate → resize → start) to preserve quorum.
//  6. Disassociate VMs, delete all reservations, delete the CRG.
//     Cleanup errors are returned — lingering reservations incur ongoing Azure costs.
func (a *azureActions) VMResizeWithCapacityReservation(ctx context.Context, targetVMSize string) error {
	clusterRG := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	location := a.oc.Location

	// Step 1: discover master VMs and their zones.
	masterVMs, err := a.listMasterVMs(ctx, clusterRG)
	if err != nil {
		return fmt.Errorf("listing master VMs: %w", err)
	}
	if len(masterVMs) == 0 {
		return fmt.Errorf("no master VMs found in resource group %s", clusterRG)
	}
	var zones []string
	seenZones := map[string]bool{}
	for _, vm := range masterVMs {
		z := vmZone(vm)
		if z == "" {
			return fmt.Errorf("VM %s has no availability zone; capacity reservation resize requires zonal VMs", *vm.Name)
		}
		if !seenZones[z] {
			seenZones[z] = true
			zones = append(zones, z)
		}
	}

	// Step 2: create the Capacity Reservation Group with the same zones as the master VMs.
	// Azure requires the CRG to declare all zones it will serve.
	a.log.Infof("creating capacity reservation group %q in zones %v", capacityReservationGroupName, zones)
	crg, err := a.armCapacityReservationGroups.CreateOrUpdate(ctx, clusterRG, capacityReservationGroupName,
		armcompute.CapacityReservationGroup{
			Location: &location,
			Zones:    pointerutils.ToSlicePtr(zones),
		})
	if err != nil {
		return fmt.Errorf("creating capacity reservation group: %w", err)
	}

	// Step 3a: create one current-SKU reservation per zone, using each VM's actual
	// hardware SKU. This handles the case where a previous partial resize left one
	// or more masters on a different family.
	a.log.Info("creating current-SKU capacity reservations")
	for _, vm := range masterVMs {
		zone := vmZone(vm)
		if vm.Properties == nil || vm.Properties.HardwareProfile == nil || vm.Properties.HardwareProfile.VMSize == nil {
			if cleanupErr := a.cleanupReservationsAndCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after SKU read failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("VM %s has no hardware profile SKU", *vm.Name)
		}
		actualVMSize := string(*vm.Properties.HardwareProfile.VMSize)
		crName := fmt.Sprintf(currentReservationNameFmt, zone)
		a.log.Infof("creating current-SKU reservation %s (SKU %s) in zone %s", crName, actualVMSize, zone)
		err = a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, capacityReservationGroupName, crName,
			armcompute.CapacityReservation{
				Location: &location,
				SKU:      &armcompute.SKU{Name: &actualVMSize, Capacity: pointerutils.ToPtr(int64(1))},
				Zones:    []*string{pointerutils.ToPtr(zone)},
			})
		if err != nil {
			if cleanupErr := a.cleanupReservationsAndCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after current-SKU reservation failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("creating current-SKU reservation for VM %s in zone %s: %w", *vm.Name, zone, err)
		}
	}

	// Step 3b: create target-SKU reservations before associating any VM.
	// Failing here means no VM has been touched — rollback is just delete reservations + CRG.
	a.log.Infof("creating target-SKU capacity reservations for %s", targetVMSize)
	for _, vm := range masterVMs {
		zone := vmZone(vm)
		crName := fmt.Sprintf(targetReservationNameFmt, zone)
		err = a.armCapacityReservations.CreateOrUpdateAndWait(ctx, clusterRG, capacityReservationGroupName, crName,
			armcompute.CapacityReservation{
				Location: &location,
				SKU:      &armcompute.SKU{Name: &targetVMSize, Capacity: pointerutils.ToPtr(int64(1))},
				Zones:    []*string{pointerutils.ToPtr(zone)},
			})
		if err != nil {
			if cleanupErr := a.cleanupReservationsAndCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after target-SKU reservation failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf(
				"target SKU %s has insufficient capacity in zone %s — consider choosing a different VM family: %w",
				targetVMSize, zone, err)
		}
	}

	// Step 4: associate all master VMs with the CRG.
	// From this point on, cleanup must disassociate VMs before deleting reservations.
	a.log.Info("associating master VMs with capacity reservation group")
	for i := range masterVMs {
		masterVMs[i].Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
			CapacityReservationGroup: &armcompute.SubResource{ID: crg.ID},
		}
		if err = a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, *masterVMs[i].Name, masterVMs[i]); err != nil {
			if cleanupErr := a.cleanupCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after association failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("associating VM %s with capacity reservation group: %w", *masterVMs[i].Name, err)
		}
	}

	// Step 5: resize each master VM one at a time to maintain etcd quorum.
	for i := range masterVMs {
		vmName := *masterVMs[i].Name
		a.log.Infof("resizing VM %s to %s", vmName, targetVMSize)

		if err = a.armVirtualMachines.DeallocateAndWait(ctx, clusterRG, vmName); err != nil {
			if cleanupErr := a.cleanupCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after deallocate failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("deallocating VM %s: %w", vmName, err)
		}

		// Re-read to get the latest VM state after deallocate.
		masterVMs[i], err = a.armVirtualMachines.Get(ctx, clusterRG, vmName)
		if err != nil {
			if cleanupErr := a.cleanupCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after VM read failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("reading VM %s after deallocate: %w", vmName, err)
		}
		masterVMs[i].Properties.HardwareProfile.VMSize = (*armcompute.VirtualMachineSizeTypes)(&targetVMSize)

		if err = a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, masterVMs[i]); err != nil {
			if cleanupErr := a.cleanupCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after resize failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("resizing VM %s: %w", vmName, err)
		}

		if err = a.armVirtualMachines.StartAndWait(ctx, clusterRG, vmName); err != nil {
			if cleanupErr := a.cleanupCRG(ctx, clusterRG, masterVMs); cleanupErr != nil {
				a.log.Warnf("cleanup after start failure also failed: %v", cleanupErr)
			}
			return fmt.Errorf("starting VM %s after resize: %w", vmName, err)
		}
	}

	// Step 6: success path — disassociate VMs and delete all reservation resources.
	// Errors are returned: lingering reservations incur ongoing Azure costs.
	a.log.Info("resize complete, cleaning up capacity reservation resources")
	if err := a.cleanupCRG(ctx, clusterRG, masterVMs); err != nil {
		return fmt.Errorf("resize succeeded but failed to clean up capacity reservation resources (manual cleanup required to avoid ongoing costs): %w", err)
	}
	return nil
}

// listMasterVMs returns VMs in the cluster RG whose names contain "master".
func (a *azureActions) listMasterVMs(ctx context.Context, clusterRG string) ([]armcompute.VirtualMachine, error) {
	allVMs, err := a.armVirtualMachines.List(ctx, clusterRG)
	if err != nil {
		return nil, err
	}
	var masters []armcompute.VirtualMachine
	for _, vm := range allVMs {
		if vm.Name != nil && strings.Contains(*vm.Name, "master") {
			masters = append(masters, vm)
		}
	}
	return masters, nil
}

// vmZone returns the availability zone of a VM (e.g. "1", "2", "3"), or "" if the VM is non-zonal.
func vmZone(vm armcompute.VirtualMachine) string {
	if len(vm.Zones) > 0 && vm.Zones[0] != nil {
		return *vm.Zones[0]
	}
	return ""
}

// cleanupReservationsAndCRG deletes all capacity reservations (current and target) and
// the CRG. Used when VMs have NOT been associated with the CRG — no VM disassociation needed.
// Returns a joined error of all failures so the caller can decide how to handle them.
func (a *azureActions) cleanupReservationsAndCRG(ctx context.Context, clusterRG string, masterVMs []armcompute.VirtualMachine) error {
	var errs []error
	for _, vm := range masterVMs {
		zone := vmZone(vm)
		for _, nameFmt := range []string{currentReservationNameFmt, targetReservationNameFmt} {
			crName := fmt.Sprintf(nameFmt, zone)
			if err := a.armCapacityReservations.DeleteAndWait(ctx, clusterRG, capacityReservationGroupName, crName); err != nil {
				errs = append(errs, fmt.Errorf("delete reservation %s: %w", crName, err))
			}
		}
	}
	if err := a.armCapacityReservationGroups.Delete(ctx, clusterRG, capacityReservationGroupName); err != nil {
		errs = append(errs, fmt.Errorf("delete capacity reservation group: %w", err))
	}
	return errors.Join(errs...)
}

// cleanupCRG disassociates all master VMs from the CRG, then delegates to
// cleanupReservationsAndCRG to delete all reservations and the CRG.
// Used when VMs have already been associated. Returns a joined error of all failures.
func (a *azureActions) cleanupCRG(ctx context.Context, clusterRG string, masterVMs []armcompute.VirtualMachine) error {
	var errs []error
	for i := range masterVMs {
		vmName := *masterVMs[i].Name
		masterVMs[i].Properties.CapacityReservation = nil
		if err := a.armVirtualMachines.CreateOrUpdateAndWait(ctx, clusterRG, vmName, masterVMs[i]); err != nil {
			errs = append(errs, fmt.Errorf("disassociate VM %s from CRG: %w", vmName, err))
		}
	}
	if err := a.cleanupReservationsAndCRG(ctx, clusterRG, masterVMs); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
