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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcomputev7 "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armcompute "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// testCRGName is a fixed CRG name used by the white-box helper tests that pass the
// name explicitly (crgCreate/crgEnsureReservations/crgDelete take it as an argument).
const testCRGName = "aro-resize-crg-cp-test"

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

// masterVMWithInstanceView returns a VM with InstanceView statuses (for GetWithInstanceView probes).
func masterVMWithInstanceView(name, zone, sku, powerState string) armcomputev7.VirtualMachine {
	vm := masterVM(name, zone, sku)
	vm.Properties.InstanceView = &armcomputev7.VirtualMachineInstanceView{
		Statuses: []*armcomputev7.InstanceViewStatus{
			{Code: pointerutils.ToPtr(powerState)},
		},
	}
	return vm
}

// masterVMAssociatedToCRG returns a VM that already has a capacity reservation group
// association, so crgDelete's disassociation step (which only acts on associated VMs)
// runs against it.
func masterVMAssociatedToCRG(name, zone, sku, crgID string) armcomputev7.VirtualMachine {
	vm := masterVM(name, zone, sku)
	vm.Properties.CapacityReservation = &armcomputev7.CapacityReservationProfile{
		CapacityReservationGroup: &armcomputev7.SubResource{
			ID: pointerutils.ToPtr(crgID),
		},
	}
	return vm
}

// --- crgCreate tests ---

func TestCRGCreate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

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

func TestCRGCreate_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	mockCRGs.EXPECT().
		CreateOrUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(armcomputev7.CapacityReservationGroup{}, errors.New("network error"))

	_, err := a.crgCreate(context.Background(), "cluster-rg", "eastus", []string{"1"}, testCRGName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- crgEnsureReservations tests ---

func TestCRGEnsureReservations_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	// Creates one target-SKU reservation per zone.
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

// --- crgDelete tests ---

func TestCRGDelete_CorrectOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	vm := masterVMAssociatedToCRG("master-0", "1", "Standard_D16s_v3", "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/"+testCRGName)
	gomock.InOrder(
		// Step 1: zero capacity FIRST (before disassociation).
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1", gomock.Any()).Return(nil),
		// Step 2: disassociate VM — GET + PUT with capacityReservationGroup.id = null.
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(vm, nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		// Step 3: delete reservation (capacity already 0).
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1").Return(nil),
		// Step 4: delete CRG last.
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", testCRGName).Return(nil),
	)

	err := a.crgDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, []string{"master-0"}, testCRGName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGDelete_NoVMs_SkipsDisassociation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	// No VM Get or UpdateAndWait calls expected.
	mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1", gomock.Any()).Return(nil)
	mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1").Return(nil)
	mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", testCRGName).Return(nil)

	err := a.crgDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, nil, testCRGName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCRGDelete_ContinuesOnPartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)

	// Zero capacity happens first (succeeds).
	mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1", gomock.Any()).Return(nil)
	// GET + PUT for disassociation: GET succeeds, PUT fails — cleanup still continues.
	mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVMAssociatedToCRG("master-0", "1", "Standard_D16s_v3", "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/"+testCRGName), nil)
	mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(errors.New("put failed"))
	mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", testCRGName, "cr-target-z1").Return(nil)
	mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", testCRGName).Return(nil)

	err := a.crgDelete(context.Background(), "cluster-rg", "eastus", "Standard_D16s_v3", []string{"1"}, []string{"master-0"}, testCRGName)
	if err == nil {
		t.Fatal("expected error from partial failure, got nil")
	}
}

// --- error classifier tests ---

func TestIsReferencedByVMError(t *testing.T) {
	referencedByVMErr := func(code string) error {
		return fmt.Errorf("the capacity reservation cannot be deleted as it is still being referenced by virtual machine(s): %w",
			&azcore.ResponseError{ErrorCode: code, StatusCode: http.StatusConflict})
	}
	unrelatedErr := func(code string) error {
		return fmt.Errorf("operation not allowed due to policy restriction: %w",
			&azcore.ResponseError{ErrorCode: code, StatusCode: http.StatusConflict})
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "OperationNotAllowed with virtual machine message",
			err:  referencedByVMErr("OperationNotAllowed"),
			want: true,
		},
		{
			name: "OperationNotAllowed without virtual machine message (policy restriction)",
			err:  unrelatedErr("OperationNotAllowed"),
			want: false,
		},
		{
			name: "different error code with virtual machine message",
			err:  referencedByVMErr("SomeOtherCode"),
			want: false,
		},
		{
			name: "OperationNotAllowed with virtual machine message but wrong status code",
			err: fmt.Errorf("virtual machine: %w",
				&azcore.ResponseError{ErrorCode: "OperationNotAllowed", StatusCode: http.StatusBadRequest}),
			want: false,
		},
		{
			name: "non-ResponseError",
			err:  errors.New("virtual machine error"),
			want: false,
		},
		{
			// Documents that the check is case-insensitive — Azure may capitalise differently.
			name: "OperationNotAllowed with mixed-case Virtual Machine message",
			err: fmt.Errorf("referenced by Virtual Machine(s): %w",
				&azcore.ResponseError{ErrorCode: "OperationNotAllowed", StatusCode: http.StatusConflict}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReferencedByVMError(tt.err)
			if got != tt.want {
				t.Errorf("isReferencedByVMError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNestedResourcesError(t *testing.T) {
	nestedResourcesErr := func(code string) error {
		return fmt.Errorf("cannot delete resource because it has nested resources: %w",
			&azcore.ResponseError{ErrorCode: code, StatusCode: http.StatusConflict})
	}
	unrelatedErr := func(code string) error {
		return fmt.Errorf("cannot delete resource due to a resource lock: %w",
			&azcore.ResponseError{ErrorCode: code, StatusCode: http.StatusConflict})
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "CannotDeleteResource with nested resources message",
			err:  nestedResourcesErr("CannotDeleteResource"),
			want: true,
		},
		{
			name: "CannotDeleteResource without nested resources message (resource lock)",
			err:  unrelatedErr("CannotDeleteResource"),
			want: false,
		},
		{
			name: "different error code with nested resources message",
			err:  nestedResourcesErr("SomeOtherCode"),
			want: false,
		},
		{
			name: "CannotDeleteResource with nested resources message but wrong status code",
			err: fmt.Errorf("nested resources: %w",
				&azcore.ResponseError{ErrorCode: "CannotDeleteResource", StatusCode: http.StatusBadRequest}),
			want: false,
		},
		{
			name: "non-ResponseError",
			err:  errors.New("nested resources error"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNestedResourcesError(tt.err)
			if got != tt.want {
				t.Errorf("isNestedResourcesError() = %v, want %v", got, tt.want)
			}
		})
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

// --- delete retry / orphan tests ---

// TestDeleteOrphanedCRG_AllRetriesExhausted verifies that when every delete attempt returns
// 404, deleteOrphanedCRG exits after exactly crgOrphanProbeRetries and returns nil
// (the CRG was never created — eventual-consistency window fully elapsed with no CRG).
func TestDeleteOrphanedCRG_AllRetriesExhausted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orig := crgOrphanProbeInterval
	crgOrphanProbeInterval = time.Millisecond
	defer func() { crgOrphanProbeInterval = orig }()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	notFoundErr := &azcore.ResponseError{ErrorCode: "ResourceNotFound", StatusCode: http.StatusNotFound}

	// Expect exactly crgOrphanProbeRetries Delete calls — all return 404.
	mockCRGs.EXPECT().
		Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").
		Return(notFoundErr).
		Times(crgOrphanProbeRetries)

	err := a.deleteOrphanedCRG(context.Background(), "cluster-rg", "aro-resize-crg-master-0")
	if err != nil {
		t.Errorf("expected nil (CRG never created), got: %v", err)
	}
}

// TestDeleteOrphanedCRG_NonNotFoundFallsBackToRetryLoop verifies that a non-404 error
// on the first orphan probe attempt (e.g. 409 CannotDeleteResource) causes
// deleteOrphanedCRG to fall back to deleteCRGWithRetry, which immediately returns
// the error for non-retryable codes.
func TestDeleteOrphanedCRG_NonNotFoundFallsBackToRetryLoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	conflictErr := &azcore.ResponseError{ErrorCode: "SomeOtherConflict", StatusCode: http.StatusConflict}

	gomock.InOrder(
		// First orphan probe returns a non-404 error (CRG exists but can't be deleted yet).
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").Return(conflictErr),
		// deleteCRGWithRetry is invoked; the error is not isNestedResourcesError so it exits immediately.
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").Return(conflictErr),
	)

	err := a.deleteOrphanedCRG(context.Background(), "cluster-rg", "aro-resize-crg-master-0")
	if err == nil {
		t.Fatal("expected error from non-404 fallback, got nil")
	}
}

// TestDeleteOrphanedCRG_ImmediateSuccess verifies that when the first Delete succeeds
// (CRG was created and is immediately visible), deleteOrphanedCRG returns nil without
// any retry delays.
func TestDeleteOrphanedCRG_ImmediateSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	mockCRGs.EXPECT().
		Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").
		Return(nil).
		Times(1)

	err := a.deleteOrphanedCRG(context.Background(), "cluster-rg", "aro-resize-crg-master-0")
	if err != nil {
		t.Errorf("expected nil on immediate success, got: %v", err)
	}
}

// TestDeleteReservationWithRetry_RetryThenSuccess verifies that deleteReservationWithRetry
// retries on a 409 "OperationNotAllowed" / "virtual machine" error and succeeds on the
// second attempt, exercising the full retry path.
func TestDeleteReservationWithRetry_RetryThenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	origInterval := crgRetryInterval
	crgRetryInterval = time.Millisecond
	defer func() { crgRetryInterval = origInterval }()

	a, _, _, mockCRs := newTestAzureActions(t, ctrl)

	// Build a retryable 409 error whose Error() string contains "virtual machine".
	retryableErr := fmt.Errorf("delete: %w: virtual machine is still associated",
		&azcore.ResponseError{ErrorCode: "OperationNotAllowed", StatusCode: http.StatusConflict})

	gomock.InOrder(
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0", "aro-resize-cr-master-0").Return(retryableErr),
		mockCRs.EXPECT().DeleteAndWait(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0", "aro-resize-cr-master-0").Return(nil),
	)

	err := a.deleteReservationWithRetry(context.Background(), "cluster-rg", "aro-resize-cr-master-0", "aro-resize-crg-master-0")
	if err != nil {
		t.Errorf("expected nil after successful retry, got: %v", err)
	}
}

// TestDeleteCRGWithRetry_RetryThenSuccess verifies that deleteCRGWithRetry retries on a
// 409 "CannotDeleteResource" / "nested resources" error and succeeds on the second attempt,
// exercising the full retry path.
func TestDeleteCRGWithRetry_RetryThenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	origInterval := crgRetryInterval
	crgRetryInterval = time.Millisecond
	defer func() { crgRetryInterval = origInterval }()

	a, _, mockCRGs, _ := newTestAzureActions(t, ctrl)

	// Build a retryable 409 error whose Error() string contains "nested resources".
	retryableErr := fmt.Errorf("delete: %w: nested resources are still present",
		&azcore.ResponseError{ErrorCode: "CannotDeleteResource", StatusCode: http.StatusConflict})

	gomock.InOrder(
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").Return(retryableErr),
		mockCRGs.EXPECT().Delete(gomock.Any(), "cluster-rg", "aro-resize-crg-master-0").Return(nil),
	)

	err := a.deleteCRGWithRetry(context.Background(), "cluster-rg", "aro-resize-crg-master-0")
	if err != nil {
		t.Errorf("expected nil after successful retry, got: %v", err)
	}
}

// --- retryOnAzureEventualConsistency tests ---

// TestRetryOnAzureEventualConsistency covers the shared retry helper directly: success
// without retry, retry-then-success, immediate return on a non-retryable error,
// exhaustion returning the last error, and context cancellation while waiting.
func TestRetryOnAzureEventualConsistency(t *testing.T) {
	a := &azureActions{log: logrus.NewEntry(logrus.StandardLogger())}
	always := func(error) bool { return true }
	retryErr := errors.New("retryable")
	fatalErr := errors.New("fatal")

	t.Run("success on first attempt", func(t *testing.T) {
		calls := 0
		err := a.retryOnAzureEventualConsistency(context.Background(), 3, time.Millisecond, func() error {
			calls++
			return nil
		}, always)
		if err != nil || calls != 1 {
			t.Errorf("got err=%v calls=%d, want nil and 1 call", err, calls)
		}
	})

	t.Run("retry then success", func(t *testing.T) {
		calls := 0
		err := a.retryOnAzureEventualConsistency(context.Background(), 5, time.Millisecond, func() error {
			calls++
			if calls < 3 {
				return retryErr
			}
			return nil
		}, always)
		if err != nil || calls != 3 {
			t.Errorf("got err=%v calls=%d, want nil and 3 calls", err, calls)
		}
	})

	t.Run("non-retryable returns immediately", func(t *testing.T) {
		calls := 0
		err := a.retryOnAzureEventualConsistency(context.Background(), 5, time.Millisecond, func() error {
			calls++
			return fatalErr
		}, func(err error) bool { return errors.Is(err, retryErr) })
		if !errors.Is(err, fatalErr) || calls != 1 {
			t.Errorf("got err=%v calls=%d, want fatal and 1 call", err, calls)
		}
	})

	t.Run("exhaustion returns last error", func(t *testing.T) {
		calls := 0
		err := a.retryOnAzureEventualConsistency(context.Background(), 3, time.Millisecond, func() error {
			calls++
			return retryErr
		}, always)
		if !errors.Is(err, retryErr) || calls != 3 {
			t.Errorf("got err=%v calls=%d, want retryable and 3 calls", err, calls)
		}
	})

	t.Run("context cancelled while waiting", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := a.retryOnAzureEventualConsistency(ctx, 5, time.Hour, func() error {
			return retryErr
		}, always)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got err=%v, want context.Canceled", err)
		}
	})
}

// --- CRGSetupForResize tests ---

func TestCRGSetupForResize_ZoneCapacityCounting(t *testing.T) {
	// Two masters in zone 1, one master in zone 2.
	// crgEnsureReservations must be called with capacity=2 for zone 1 and capacity=1 for zone 2.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, mockCRGs, mockCRs := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	// Three VMs: master-2 and master-0 in zone 1, master-1 in zone 2.
	gomock.InOrder(
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-2").Return(masterVM("master-2", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-1").Return(masterVM("master-1", "2", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		// crgCreate receives deduplicated, sorted zones: ["1", "2"].
		mockCRGs.EXPECT().CreateOrUpdate(gomock.Any(), "cluster-rg", gomock.Any(),
			gomock.AssignableToTypeOf(armcomputev7.CapacityReservationGroup{})).
			Return(armcomputev7.CapacityReservationGroup{ID: pointerutils.ToPtr(crgID)}, nil),
		// Zone "1": capacity=2 (two VMs in zone 1).
		mockCRs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", gomock.Any(), "cr-target-z1",
			gomock.AssignableToTypeOf(armcomputev7.CapacityReservation{})).
			DoAndReturn(func(_ context.Context, _, _, _ string, cr armcomputev7.CapacityReservation) error {
				if cr.SKU == nil || cr.SKU.Capacity == nil || *cr.SKU.Capacity != 2 {
					t.Errorf("zone 1: expected capacity=2, got %v", cr.SKU.Capacity)
				}
				return nil
			}),
		// Zone "2": capacity=1.
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
	gotID, gotName, gotZones, err := a.CRGSetupForResize(context.Background(), vmNames, targetSKU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != crgID {
		t.Errorf("expected crgID=%s, got %s", crgID, gotID)
	}
	if gotName == "" {
		t.Error("expected non-empty crgName")
	}
	// Returned zones must be deduplicated and sorted.
	if len(gotZones) != 2 || gotZones[0] != "1" || gotZones[1] != "2" {
		t.Errorf("expected zones=[1 2], got %v", gotZones)
	}
}

// --- VMResizeWithCRG tests ---

func TestVMResizeWithCRG_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
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

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
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

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"
	resizeErr := errors.New("resize failed")

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
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

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
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

	a, mockVMs, _, _ := newTestAzureActions(t, ctrl)
	const crgID = "/subscriptions/sub/resourceGroups/cluster-rg/providers/Microsoft.Compute/capacityReservationGroups/test-crg"
	const targetSKU = "Standard_D16s_v5"
	startErr := errors.New("start failed")

	gomock.InOrder(
		mockVMs.EXPECT().DeallocateAndWait(gomock.Any(), "cluster-rg", "master-0").Return(nil),
		mockVMs.EXPECT().Get(gomock.Any(), "cluster-rg", "master-0").Return(masterVM("master-0", "1", "Standard_D8s_v3"), nil),
		mockVMs.EXPECT().CreateOrUpdateAndWait(gomock.Any(), "cluster-rg", "master-0", gomock.Any()).Return(nil),
		mockVMs.EXPECT().StartAndWait(gomock.Any(), "cluster-rg", "master-0").Return(startErr),
		mockVMs.EXPECT().GetWithInstanceView(gomock.Any(), "cluster-rg", "master-0").
			Return(masterVMWithInstanceView("master-0", "1", targetSKU, "PowerState/deallocated"), nil),
	)

	err := a.VMResizeWithCRG(context.Background(), "master-0", crgID, targetSKU)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, startErr) {
		t.Errorf("expected error to wrap startErr, got: %v", err)
	}
}
