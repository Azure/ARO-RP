package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcomputev7 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// withFastRetryBackoff overrides arm.TransientBackoff to near-instant for tests exercising the retry path.
func withFastRetryBackoff(t *testing.T) {
	t.Helper()
	orig := arm.TransientBackoff
	arm.TransientBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
	t.Cleanup(func() { arm.TransientBackoff = orig })
}

func retryable429() error {
	return &azcore.ResponseError{StatusCode: http.StatusTooManyRequests}
}

// testCRGName is a fixed name for white-box helper tests (crgCreate/crgEnsureReservations take it as an argument).
const testCRGName = "aro-resize-crg-cp-test"

func newTestAzureActions(t *testing.T, ctrl *gomock.Controller) (
	*azureActions,
	*mock_armcompute.MockVirtualMachinesClient,
	*mock_armcompute.MockCapacityReservationGroupsClient,
	*mock_armcompute.MockCapacityReservationsClient,
	*mock_armcompute.MockResourceSKUsClient,
) {
	t.Helper()
	mockVMs := mock_armcompute.NewMockVirtualMachinesClient(ctrl)
	mockCRGs := mock_armcompute.NewMockCapacityReservationGroupsClient(ctrl)
	mockCRs := mock_armcompute.NewMockCapacityReservationsClient(ctrl)
	mockSKUs := mock_armcompute.NewMockResourceSKUsClient(ctrl)

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
		resourceSkus:                 mockSKUs,
	}
	return a, mockVMs, mockCRGs, mockCRs, mockSKUs
}

func skuIter(skus ...*armcomputev7.ResourceSKU) iter.Seq2[*armcomputev7.ResourceSKU, error] {
	return func(yield func(*armcomputev7.ResourceSKU, error) bool) {
		for _, sku := range skus {
			if !yield(sku, nil) {
				return
			}
		}
	}
}

func unrestrictedSKU(name string) *armcomputev7.ResourceSKU {
	return &armcomputev7.ResourceSKU{
		Name:         pointerutils.ToPtr(name),
		ResourceType: pointerutils.ToPtr("virtualMachines"),
		Locations:    []*string{pointerutils.ToPtr("eastus")},
		LocationInfo: []*armcomputev7.ResourceSKULocationInfo{
			{Location: pointerutils.ToPtr("eastus")},
		},
	}
}

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

func masterVMWithInstanceView(name, zone, sku, powerState string) armcomputev7.VirtualMachine {
	vm := masterVM(name, zone, sku)
	vm.Properties.InstanceView = &armcomputev7.VirtualMachineInstanceView{
		Statuses: []*armcomputev7.InstanceViewStatus{
			{Code: pointerutils.ToPtr(powerState)},
		},
	}
	return vm
}

func TestCRGCreate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _, _ := newTestAzureActions(t, ctrl)

	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{ID: &crgID}, nil)

	got, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1", "2", "3"}, testCRGName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != crgID {
		t.Errorf("got CRG ID %q, want %q", got, crgID)
	}
}

func TestCRGCreate_RetriesTransientErrorThenSucceeds(t *testing.T) {
	// Proves arm.Retryable inside crgCreate retries a transient 429 and succeeds on the second attempt.
	withFastRetryBackoff(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _, _ := newTestAzureActions(t, ctrl)

	crgID := "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	gomock.InOrder(
		mockCRGs.EXPECT().
			CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{}, retryable429()),
		mockCRGs.EXPECT().
			CreateOrUpdate(gomock.Any(), "cluster-rg", testCRGName, gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: &crgID}, nil),
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

	a, _, mockCRGs, _, _ := newTestAzureActions(t, ctrl)

	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, errors.New("network error"))

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCRGCreate_AuthorizationFailed_ReturnsActionableError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _, _ := newTestAzureActions(t, ctrl)

	authErr := &azcore.ResponseError{
		ErrorCode:  "AuthorizationFailed",
		StatusCode: http.StatusForbidden,
	}
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, authErr)

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Errorf("expected error to wrap the auth error, got: %v", err)
	}
}

func TestCRGCreate_QuotaExceeded_ReturnsActionableError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _, _ := newTestAzureActions(t, ctrl)

	quotaErr := &azcore.ResponseError{
		ErrorCode:  "QuotaExceeded",
		StatusCode: http.StatusConflict,
	}
	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, quotaErr)

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, quotaErr) {
		t.Errorf("expected error to wrap the quota error, got: %v", err)
	}
	if !isQuotaError(err) {
		t.Errorf("expected returned error to be recognised as a quota error, got: %v", err)
	}
}

func TestCRGEnsureReservations_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs, _ := newTestAzureActions(t, ctrl)

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

	a, _, _, mockCRs, _ := newTestAzureActions(t, ctrl)

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
	if !isCapacityError(err) {
		t.Errorf("expected returned error to be recognised as a capacity error, got: %v", err)
	}
}

func TestCRGEnsureReservations_TargetFails_ReturnsQuotaError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs, _ := newTestAzureActions(t, ctrl)

	quotaErr := &azcore.ResponseError{
		ErrorCode:  "QuotaExceeded",
		StatusCode: http.StatusConflict,
	}
	mockCRs.EXPECT().
		CreateOrUpdateAndWait(gomock.Any(), gomock.Any(), gomock.Any(), "cr-target-z1", gomock.Any()).
		Return(quotaErr)

	err := a.crgEnsureReservations(context.Background(), "cluster-rg", "eastus", "1", "Standard_D16s_v3", testCRGName, 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isQuotaError(err) {
		t.Errorf("expected returned error to be recognised as a quota error, got: %v", err)
	}
}

func TestCRGEnsureReservations_AuthorizationFailed_ReturnsActionableError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs, _ := newTestAzureActions(t, ctrl)

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
	// Two masters in zone 1, one master in zone 2 — capacity reservation must reflect each count.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs, mockSKUs := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-2").Return(masterVM("master-2", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockSKUs.EXPECT().List(gomock.Any(), "location eq eastus", false).Return(skuIter(unrestrictedSKU(targetSKU))),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(),
			gomock.AssignableToTypeOf(armcomputev7.CapacityReservationGroup{})).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z1",
			gomock.AssignableToTypeOf(armcomputev7.CapacityReservation{})).
			DoAndReturn(func(_ context.Context, _, _, _ string, cr armcomputev7.CapacityReservation) error {
				if cr.SKU == nil || cr.SKU.Capacity == nil || *cr.SKU.Capacity != 2 {
					t.Errorf("zone 1: expected capacity=2, got %v", cr.SKU.Capacity)
				}
				return nil
			}),
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z2",
			gomock.AssignableToTypeOf(armcomputev7.CapacityReservation{})).
			DoAndReturn(func(_ context.Context, _, _, _ string, cr armcomputev7.CapacityReservation) error {
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
	// master-0 is already at targetSKU — must be excluded from zone sizing and from resizeVMNames.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs, mockSKUs := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", targetSKU), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockSKUs.EXPECT().List(gomock.Any(), "location eq eastus", false).Return(skuIter(unrestrictedSKU(targetSKU))),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(), gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
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

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
	// Zone 2's reservation fails; no cleanup of the already-created CRG or zone-1 reservation is expected.
	// gomock strict mode enforces this: an unexpected Delete/DeleteAndWait call would fail the test.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs, mockSKUs := newTestAzureActions(t, ctrl)
	const targetSKU = "Standard_D16s_v5"
	reservationErr := errors.New("reservation create failed")

	gomock.InOrder(
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockSKUs.EXPECT().List(gomock.Any(), "location eq eastus", false).Return(skuIter(unrestrictedSKU(targetSKU))),
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(), gomock.Any()).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr("crg-id")}, nil),
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
	// Proves arm.Retryable around DeallocateAndWait retries a 429 and succeeds on the second attempt.
	withFastRetryBackoff(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
	// Pre-existing CRG association on the VM must be overwritten; it must not be treated as an error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const staleCRGID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/stale-crg"
	const targetSKU = "Standard_D16s_v5"

	vmWithStaleAssociation := masterVM("master-0", "1", "Standard_D8s_v3")
	vmWithStaleAssociation.Properties.CapacityReservation = &armcomputev7.CapacityReservationProfile{
		CapacityReservationGroup: &armcomputev7.SubResource{ID: pointerutils.ToPtr(staleCRGID)},
	}

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().GetDefault(gomock.Any(), "cluster-rg", "master-0").Return(vmWithStaleAssociation, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _ string, vm armcomputev7.VirtualMachine) error {
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
	// PUT times out, but GetWithInstanceView confirms VM is at target SKU and running.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
	// PUT errors, GetWithInstanceView shows VM is still at old SKU — return the error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
	// StartAndWait returns a transient error, but GetWithInstanceView shows VM is running.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
	// StartAndWait errors and GetWithInstanceView shows VM is NOT running — return error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _, _ := newTestAzureActions(t, ctrl)
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
