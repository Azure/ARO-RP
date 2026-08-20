package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func withFastRetryBackoff(t *testing.T) {
	t.Helper()
	orig := arm.TransientBackoff
	arm.TransientBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
	t.Cleanup(func() { arm.TransientBackoff = orig })
}

func retryable429() error {
	return &azcore.ResponseError{StatusCode: http.StatusTooManyRequests}
}

const testCRGName = "aro-resize-crg-cp-test"

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

func masterVM(name, zone, sku string) armcompute.VirtualMachine {
	return armcompute.VirtualMachine{
		Name:  pointerutils.ToPtr(name),
		Zones: []*string{pointerutils.ToPtr(zone)},
		Properties: &armcompute.VirtualMachineProperties{
			HardwareProfile: &armcompute.HardwareProfile{
				VMSize: (*armcompute.VirtualMachineSizeTypes)(pointerutils.ToPtr(sku)),
			},
		},
	}
}

func masterVMWithInstanceView(name, zone, sku, powerState string) armcompute.VirtualMachine {
	vm := masterVM(name, zone, sku)
	vm.Properties.InstanceView = &armcompute.VirtualMachineInstanceView{
		Statuses: []*armcompute.InstanceViewStatus{
			{Code: pointerutils.ToPtr(powerState)},
		},
	}
	return vm
}

func TestCRGCreate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
		Return(armcompute.CapacityReservationGroup{ID: &crgID}, nil)

	got, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1", "2", "3"}, testCRGName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != crgID {
		t.Errorf("got CRG ID %q, want %q", got, crgID)
	}
}

func TestCRGCreate_RetriesTransientErrorThenSucceeds(t *testing.T) {
	withFastRetryBackoff(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	gomock.InOrder(
		mockCRGs.EXPECT().
			CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
			Return(armcompute.CapacityReservationGroup{}, retryable429()),
		mockCRGs.EXPECT().
			CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
			Return(armcompute.CapacityReservationGroup{ID: &crgID}, nil),
	)

	got, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
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
		Return(armcompute.CapacityReservationGroup{}, errors.New("network error"))

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCRGCreate_AuthorizationFailed_ReturnsActionableError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	authErr := &azcore.ResponseError{
		ErrorCode:  "AuthorizationFailed",
		StatusCode: http.StatusForbidden,
	}
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcompute.CapacityReservationGroup{}, authErr)

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Errorf("expected error to wrap the auth error, got: %v", err)
	}
}

func TestCRGEnsureReservations_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	mockCRs.EXPECT().
		CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1", gomock.Any()).
		Return(nil)

	err := a.crgEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3", testCRGName, 1)
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

	err := a.crgEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3", testCRGName, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Assert against the returned (wrapped) error, not the raw mock error, to verify that
	// crgEnsureReservations correctly wraps the underlying *azcore.ResponseError so that
	// isCapacityError (which uses errors.As) can still unwrap and recognise it.
	if !isCapacityError(err) {
		t.Errorf("expected returned error to be recognised as a capacity error, got: %v", err)
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

	err := a.crgEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3", testCRGName, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Errorf("expected error to wrap the auth error, got: %v", err)
	}
}

func TestIsCapacityError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "AllocationFailed",
			err:  &azcore.ResponseError{ErrorCode: "AllocationFailed", StatusCode: http.StatusConflict},
			want: true,
		},
		{
			name: "OverconstrainedAllocationRequest",
			err:  &azcore.ResponseError{ErrorCode: "OverconstrainedAllocationRequest", StatusCode: http.StatusConflict},
			want: true,
		},
		{
			name: "CapacityReservationCapacityExceeded",
			err:  &azcore.ResponseError{ErrorCode: "CapacityReservationCapacityExceeded", StatusCode: http.StatusConflict},
			want: true,
		},
		{
			name: "AllocationFailed wrapped in fmt.Errorf",
			err:  fmt.Errorf("wrapping: %w", &azcore.ResponseError{ErrorCode: "AllocationFailed", StatusCode: http.StatusConflict}),
			want: true,
		},
		{
			name: "unrelated error code",
			err:  &azcore.ResponseError{ErrorCode: "AuthorizationFailed", StatusCode: http.StatusForbidden},
			want: false,
		},
		{
			name: "non-ResponseError",
			err:  errors.New("allocation failed"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCapacityError(tt.err)
			if got != tt.want {
				t.Errorf("isCapacityError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCRGSetupForResize_ZoneCapacityCounting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-2").Return(masterVM("master-2", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(),
			gomock.AssignableToTypeOf(armcompute.CapacityReservationGroup{})).
			Return(armcompute.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z1",
			gomock.AssignableToTypeOf(armcompute.CapacityReservation{})).
			DoAndReturn(func(_ context.Context, _, _, _ string, cr armcompute.CapacityReservation) error {
				if cr.SKU == nil || cr.SKU.Capacity == nil || *cr.SKU.Capacity != 2 {
					t.Errorf("zone 1: expected capacity=2, got %v", cr.SKU.Capacity)
				}
				return nil
			}),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z2",
			gomock.AssignableToTypeOf(armcompute.CapacityReservation{})).
			DoAndReturn(func(_ context.Context, _, _, _ string, cr armcompute.CapacityReservation) error {
				if cr.SKU == nil || cr.SKU.Capacity == nil || *cr.SKU.Capacity != 1 {
					t.Errorf("zone 2: expected capacity=1, got %v", cr.SKU.Capacity)
				}
				return nil
			}),
	)

	vmNames := []string{"master-2", "master-1", "master-0"}
	gotID, gotName, gotZones, gotResizeVMNames, err := a.CRGSetupForResize(context.Background(), vmNames, targetSKU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != crgID {
		t.Errorf("expected crgID=%s, got %s", crgID, gotID)
	}
	if gotName == "" {
		t.Error("expected non-empty crgName")
	}
	if len(gotZones) != 2 || gotZones[0] != "1" || gotZones[1] != "2" {
		t.Errorf("expected zones=[1 2], got %v", gotZones)
	}
	if len(gotResizeVMNames) != 3 {
		t.Errorf("expected 3 VMs to need resizing, got %v", gotResizeVMNames)
	}
}

func TestCRGSetupForResize_ExcludesVMsAlreadyAtTargetSKU(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", targetSKU), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(), gomock.Any()).
			Return(armcompute.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z2", gomock.Any()).
			Return(nil),
	)

	vmNames := []string{"master-0", "master-1"}
	_, _, gotZones, gotResizeVMNames, err := a.CRGSetupForResize(context.Background(), vmNames, targetSKU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotZones) != 1 || gotZones[0] != "2" {
		t.Errorf("expected zones=[2] (zone 1 excluded), got %v", gotZones)
	}
	if len(gotResizeVMNames) != 1 || gotResizeVMNames[0] != "master-1" {
		t.Errorf("expected resizeVMNames=[master-1] (master-0 excluded), got %v", gotResizeVMNames)
	}
}

func TestCRGSetupForResize_AllVMsAlreadyAtTargetSKU_NoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const targetSKU = "Standard_D16s_v5"

	mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", targetSKU), nil)

	gotID, gotName, gotZones, gotResizeVMNames, err := a.CRGSetupForResize(context.Background(), []string{"master-0"}, targetSKU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "" || gotName != "" || gotZones != nil || gotResizeVMNames != nil {
		t.Errorf("expected no CRG to be created, got id=%q name=%q zones=%v resizeVMNames=%v", gotID, gotName, gotZones, gotResizeVMNames)
	}
}

func TestCRGSetupForResize_ReservationFails_ReturnsErrorWithoutCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	const targetSKU = "Standard_D16s_v5"
	reservationErr := errors.New("reservation create failed")

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(), gomock.Any()).
			Return(armcompute.CapacityReservationGroup{ID: pointerutils.ToPtr("crg-id")}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z1", gomock.Any()).Return(nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z2", gomock.Any()).Return(reservationErr),
	)

	_, _, _, _, err := a.CRGSetupForResize(context.Background(), []string{"master-0", "master-1"}, targetSKU)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, reservationErr) {
		t.Errorf("expected error to wrap the underlying reservation error, got: %v", err)
	}
}

func TestVMResizeWithCRG_DeallocateRetriesTransientErrorThenSucceeds(t *testing.T) {
	withFastRetryBackoff(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(retryable429()),
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
	)

	if err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVMResizeWithCRG_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
	)

	if err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVMResizeWithCRG_WarnsOnExistingCRGAssociation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const staleCRGID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/stale-crg"
	const targetSKU = "Standard_D16s_v5"

	vmWithStaleAssociation := masterVM("master-0", "1", "Standard_D8s_v3")
	vmWithStaleAssociation.Properties.CapacityReservation = &armcompute.CapacityReservationProfile{
		CapacityReservationGroup: &armcompute.SubResource{ID: pointerutils.ToPtr(staleCRGID)},
	}

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(vmWithStaleAssociation, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _ string, vm armcompute.VirtualMachine) error {
				if vm.Properties.CapacityReservation == nil || vm.Properties.CapacityReservation.CapacityReservationGroup == nil ||
					vm.Properties.CapacityReservation.CapacityReservationGroup.ID == nil ||
					*vm.Properties.CapacityReservation.CapacityReservationGroup.ID != crgID {
					t.Errorf("expected VM to be associated with the new CRG %s, got %v", crgID, vm.Properties.CapacityReservation)
				}
				return nil
			}),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
	)

	if err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVMResizeWithCRG_ResizePUT_TransientError_VMRunningAtTargetSKU_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).
			Return(errors.New("transient poller error")),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetWithInstanceView(gomock.Any(), "cluster-rg", "master-0").
			Return(masterVMWithInstanceView("master-0", "1", targetSKU, "PowerState/running"), nil),
	)

	if err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU); err != nil {
		t.Fatalf("expected nil (transient error resolved), got: %v", err)
	}
}

func TestVMResizeWithCRG_ResizePUT_TransientError_VMAtOldSKU_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"
	resizeErr := errors.New("resize failed")

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(resizeErr),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetWithInstanceView(gomock.Any(), "cluster-rg", "master-0").
			Return(masterVMWithInstanceView("master-0", "1", "Standard_D8s_v3", "PowerState/running"), nil),
	)

	err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, resizeErr) {
		t.Errorf("expected error to wrap resizeErr, got: %v", err)
	}
}

func TestVMResizeWithCRG_StartFails_VMRunning_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").
			Return(errors.New("start poller timeout")),
		mockVMs.EXPECT().GetWithInstanceView(gomock.Any(), "cluster-rg", "master-0").
			Return(masterVMWithInstanceView("master-0", "1", targetSKU, "PowerState/running"), nil),
	)

	if err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU); err != nil {
		t.Fatalf("expected nil (VM is running), got: %v", err)
	}
}

func TestVMResizeWithCRG_StartFails_VMNotRunning_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"
	startErr := errors.New("start failed")

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(startErr),
		mockVMs.EXPECT().GetWithInstanceView(gomock.Any(), "cluster-rg", "master-0").
			Return(masterVMWithInstanceView("master-0", "1", targetSKU, "PowerState/deallocated"), nil),
		// VM is not running, so a best-effort restart is attempted before the error is returned.
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(startErr),
	)

	err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, startErr) {
		t.Errorf("expected error to wrap startErr, got: %v", err)
	}
}
