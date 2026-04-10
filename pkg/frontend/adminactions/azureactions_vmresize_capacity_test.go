package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	armcomputev7 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// newTestAzureActions builds an azureActions with mock ARM clients injected.
func newTestAzureActions(t *testing.T, ctrl *gomock.Controller) (
	*azureActions,
	*mock_armcompute.MockVirtualMachinesClient,
	*mock_armcompute.MockCapacityReservationGroupsClient,
	*mock_armcompute.MockCapacityReservationsClient,
) {
	t.Helper()
	mockVMs := mock_armcompute.NewMockVirtualMachinesClient(ctrl)
	mockCRGs := mock_armcompute.NewMockCapacityReservationGroupsClient(ctrl)
	mockCRs := mock_armcompute.NewMockCapacityReservationsClient(ctrl)

	a := &azureActions{
		log: logrus.NewEntry(logrus.StandardLogger()),
		oc: &api.OpenShiftCluster{
			Location: "eastus",
			Properties: api.OpenShiftClusterProperties{
				ClusterProfile: api.ClusterProfile{
					ResourceGroupID: "/subscriptions/sub/resourceGroups/cluster-rg",
				},
			},
		},
		armVirtualMachines:           mockVMs,
		armCapacityReservationGroups: mockCRGs,
		armCapacityReservations:      mockCRs,
	}
	return a, mockVMs, mockCRGs, mockCRs
}

// masterVM builds a minimal master VirtualMachine for tests.
func masterVM(name, zone, sku string) armcomputev7.VirtualMachine {
	return armcomputev7.VirtualMachine{
		Name:  pointerutils.ToPtr(name),
		Zones: []*string{pointerutils.ToPtr(zone)},
		Properties: &armcomputev7.VirtualMachineProperties{
			HardwareProfile: &armcomputev7.HardwareProfile{
				VMSize: (*armcomputev7.VirtualMachineSizeTypes)(pointerutils.ToPtr(sku)),
			},
		},
	}
}

// --- CRGCreate tests ---

func TestCRGCreate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{ID: &crgID}, nil)

	got, err := a.CRGCreate(context.Background(), "cluster-rg", "eastus", []string{"1", "2", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != crgID {
		t.Errorf("got CRG ID %q, want %q", got, crgID)
	}
}

func TestCRGCreate_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, errors.New("network error"))

	_, err := a.CRGCreate(context.Background(), "cluster-rg", "eastus", []string{"1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CRGEnsureReservations tests ---

func TestCRGEnsureReservations_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	// Creates one target-SKU reservation per zone.
	mockCRs.EXPECT().
		CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).
		Return(nil)

	err := a.CRGEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGEnsureReservations_TargetFails_ReturnsCapacityError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	capacityErr := &azcore.ResponseError{
		ErrorCode:  "AllocationFailed",
		StatusCode: http.StatusConflict,
	}
	mockCRs.EXPECT().
		CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), "cr-target-z1", gomock.Any()).
		Return(capacityErr)

	err := a.CRGEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isCapacityError(capacityErr) {
		t.Errorf("expected AllocationFailed to be recognized as a capacity error")
	}
}

func TestCRGEnsureReservations_AuthorizationFailed_ReturnsActionableError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	authErr := &azcore.ResponseError{
		ErrorCode:  "AuthorizationFailed",
		StatusCode: http.StatusForbidden,
	}
	mockCRs.EXPECT().
		CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), "cr-target-z1", gomock.Any()).
		Return(authErr)

	err := a.CRGEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Errorf("expected error to wrap the auth error, got: %v", err)
	}
}

// --- CRGAssociateVM tests ---

func TestCRGAssociateVM_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	vm := masterVM("master-0", "1", "Standard_D8s_v3")

	mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil)
	mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil)

	err := a.CRGAssociateVM(context.Background(), "cluster-rg", "master-0", "/subscriptions/sub/crg-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGAssociateVM_GetFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)

	mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(armcomputev7.VirtualMachine{}, errors.New("not found"))

	err := a.CRGAssociateVM(context.Background(), "cluster-rg", "master-0", "/subscriptions/sub/crg-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CRGDelete tests ---

func TestCRGDelete_CorrectOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	gomock.InOrder(
		// Step 1: zero capacity FIRST (before disassociation).
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		// Step 2: disassociate VM — GET full VM then PUT with empty SubResource.
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		// Step 3: delete reservation (capacity already 0).
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		// Step 4: delete CRG last.
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, []string{"master-0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGDelete_NoVMs_SkipsDisassociation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	// No VM Get or UpdateAndWait calls expected.
	mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil)
	mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil)
	mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil)

	err := a.CRGDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGDelete_ContinuesOnPartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	// Zero capacity happens first (succeeds).
	mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil)
	// Get fails so CreateOrUpdateAndWait for disassociation is skipped.
	mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(armcomputev7.VirtualMachine{}, errors.New("read failed"))
	mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil)
	mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil)

	err := a.CRGDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, []string{"master-0"})
	if err == nil {
		t.Fatal("expected error from partial failure, got nil")
	}
}

// --- CRGResizeSingleVM tests ---

// TestCRGResizeSingleVM_HappyPath verifies the complete correct flow in order:
// CRGCreate → EnsureReservations → Deallocate → Get+Resize+Associate →
// Start → CRGDelete (zero+disassociate+delete+deleteCRG).
func TestCRGResizeSingleVM_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	vm := masterVM("master-0", "1", "Standard_D8s_v3")

	const (
		clusterRG = "cluster-rg"
		location  = "eastus"
		zone      = "1"
		targetSKU = "Standard_D16s_v3"
		crgID     = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"
	)

	gomock.InOrder(
		// Step 1: create CRG.
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), clusterRG, capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		// Step 2: reserve target capacity.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, capacityReservationGroupName, "cr-target-z1", gomock.Any()).
			Return(nil),
		// Step 3: deallocate.
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), clusterRG, "master-0").Return(nil),
		// Step 4: read + resize + associate in one call.
		mockVMs.EXPECT().Get(gomock.Any(), clusterRG, "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, "master-0", gomock.Any()).Return(nil),
		// Step 5: start VM while still associated (reservation guarantees capacity).
		mockVMs.EXPECT().StartAndWait(gomock.Any(), clusterRG, "master-0").Return(nil),
		// Step 6: CRGDelete — zero capacity FIRST (allows delete even if VM still bookmarked).
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		// Step 6: CRGDelete — disassociate VM via GET + PUT with empty SubResource.
		mockVMs.EXPECT().Get(gomock.Any(), clusterRG, "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), clusterRG, "master-0", gomock.Any()).Return(nil),
		// Step 6: CRGDelete — delete reservation.
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), clusterRG, capacityReservationGroupName, "cr-target-z1").Return(nil),
		// Step 6: CRGDelete — delete CRG.
		mockCRGs.EXPECT().Delete(gomock.Any(), clusterRG, capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), clusterRG, location, "master-0", zone, targetSKU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGResizeSingleVM_CRGCreateFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, errors.New("permission denied"))

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCRGResizeSingleVM_ReservationCreateFails_TriggersCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"

	gomock.InOrder(
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		// Reservation create fails — no VMs associated yet.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).
			Return(errors.New("quota exceeded")),
		// Cleanup: zero capacity (may fail gracefully), delete reservation, delete CRG — no VM disassociation.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCRGResizeSingleVM_ResizeFails_TriggersCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	vm := masterVM("master-0", "1", "Standard_D8s_v3")

	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"

	gomock.InOrder(
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		// Resize+associate fails — VM is unlikely to be associated, cleanup uses nil vmNames.
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(errors.New("resize failed")),
		// Cleanup: zero capacity, no VM disassociation, delete reservation, delete CRG.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCRGResizeSingleVM_DeallocateFails_TriggersCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"

	gomock.InOrder(
		// Setup succeeds.
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		// Deallocation fails.
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(errors.New("deallocation failed")),
		// Cleanup: no VMs associated yet, so no VM disassociation. Zero + delete reservation + delete CRG.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestCRGResizeSingleVM_CRGDeleteFails_AfterVMRunning verifies that when CRGDelete fails
// after the VM is already started, the error is returned (VM stays running).
func TestCRGResizeSingleVM_CRGDeleteFails_AfterVMRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	vm := masterVM("master-0", "1", "Standard_D8s_v3")

	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"

	gomock.InOrder(
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		// VM starts successfully while still associated with CRG.
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		// CRGDelete: zero capacity FIRST, then fail on disassociation.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(errors.New("put failed")),
		// Delete reservation and CRG still run even after disassociation error.
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error from CRGDelete failure, got nil")
	}
}

func TestCRGResizeSingleVM_StartFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	vm := masterVM("master-0", "1", "Standard_D8s_v3")

	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/aro-resize-crg"

	gomock.InOrder(
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", capacityReservationGroupName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		// Start fails — triggers cleanup with vmName (VM was associated).
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(errors.New("start failed")),
		// Cleanup: zero capacity FIRST, then disassociate VM, delete reservation, delete CRG.
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1", gomock.Any()).Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", capacityReservationGroupName, "cr-target-z1").Return(nil),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", capacityReservationGroupName).Return(nil),
	)

	err := a.CRGResizeSingleVM(context.Background(), "cluster-rg", "eastus", "master-0", "1", "Standard_D16s_v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
