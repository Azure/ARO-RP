package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func virtualMachineWithSize(vmSize string) mgmtcompute.VirtualMachine {
	return mgmtcompute.VirtualMachine{
		VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
			HardwareProfile: &mgmtcompute.HardwareProfile{
				VMSize: mgmtcompute.VirtualMachineSizeTypes(vmSize),
			},
		},
	}
}

func assertErrorContainsAll(t *testing.T, err error, substrs ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	for _, substr := range substrs {
		if !strings.Contains(err.Error(), substr) {
			t.Fatalf("error %q does not contain %q", err.Error(), substr)
		}
	}
}

func TestResizeControlPlaneRollback(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	running := "Running"
	desiredSize := "Standard_D16s_v5"
	clusterResourceGroupName := "test-cluster"

	t.Run("rolls back stopped and cordoned node when resize fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
			masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

		gomock.InOrder(
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
				Return(nodeJSON("master-0", true), nil),
			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-0", mgmtcompute.InstanceView).
				Return(virtualMachineWithSize("Standard_D8s_v3"), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
				Return(nodeJSON("master-0", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
			a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-0", desiredSize).Return(errors.New("Azure resize error")),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-0").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
				Return(nodeJSON("master-0", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-0", false).Return(nil),
		)

		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to resize node master-0: resizing VM: Azure resize error",
			"Steps taken:",
			"master-0:cordon",
			"master-0:drain",
			"master-0:stop",
			"master-0:resize failed",
			"Rollback:",
			"master-0:start",
			"master-0:waitReady",
			"master-0:restoreSchedulability",
		)
	})

	t.Run("rolls back previously resized node when a later node fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
			masterMachineListJSON(
				masterMachine("master-0", desiredSize, running),
				masterMachine("master-1", "Standard_D8s_v3", running),
				masterMachine("master-2", "Standard_D8s_v3", running),
			), nil)

		gomock.InOrder(
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-2", mgmtcompute.InstanceView).
				Return(virtualMachineWithSize("Standard_D8s_v3"), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-2").Return(nil),
			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", desiredSize).Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", false).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", "Standard_D8s_v3"), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-1", mgmtcompute.InstanceView).
				Return(virtualMachineWithSize("Standard_D8s_v3"), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-1", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-1").Return(errors.New("could not drain node after 3 retries: drain error")),

			k.EXPECT().CordonNode(gomock.Any(), "master-1", false).Return(nil),

			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", "Standard_D8s_v3").Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", desiredSize), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, obj any) error {
					return nil
				}),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSONWithSchedulability("master-2", true, false), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
		)

		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to resize node master-1: draining node: could not drain node after 3 retries: drain error",
			"Rollback:",
			"master-1:restoreSchedulability",
			"master-2:restoreVMSize",
			"master-2:restoreMachine",
			"master-2:restoreNodeLabels",
		)
	})

	t.Run("rechecks control plane health before the next node and rolls back prior progress", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
			masterMachineListJSON(
				masterMachine("master-0", desiredSize, running),
				masterMachine("master-1", "Standard_D8s_v3", running),
				masterMachine("master-2", "Standard_D8s_v3", running),
			), nil)

		gomock.InOrder(
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-2", mgmtcompute.InstanceView).
				Return(virtualMachineWithSize("Standard_D8s_v3"), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-2").Return(nil),
			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", desiredSize).Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", false).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", "Standard_D8s_v3"), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),

			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", false), nil),

			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", "Standard_D8s_v3").Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", desiredSize), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
		)

		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"Control plane node master-2 is not Ready",
			"Rollback:",
			"master-2:restoreVMSize",
			"master-2:restoreMachine",
			"master-2:restoreNodeLabels",
		)
	})

	t.Run("rolls back prior progress when later snapshot capture fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), "Machine", machineNamespace).Return(
			masterMachineListJSON(
				masterMachine("master-0", desiredSize, running),
				masterMachine("master-1", "Standard_D8s_v3", running),
				masterMachine("master-2", "Standard_D8s_v3", running),
			), nil)

		gomock.InOrder(
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-2", mgmtcompute.InstanceView).
				Return(virtualMachineWithSize("Standard_D8s_v3"), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-2").Return(nil),
			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", desiredSize).Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-2", false).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", "Standard_D8s_v3"), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			a.EXPECT().GetVirtualMachine(gomock.Any(), clusterResourceGroupName, "master-1", mgmtcompute.InstanceView).
				Return(mgmtcompute.VirtualMachine{}, errors.New("transient ARM read failed")),

			a.EXPECT().VMStopAndWait(gomock.Any(), "master-2", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-2", "Standard_D8s_v3").Return(nil),
			a.EXPECT().VMStartAndWait(gomock.Any(), "master-2").Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-2").
				Return(machineJSON("master-2", desiredSize), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSONWithSchedulability("master-2", true, false), nil),
			k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
		)

		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to capture Azure VM state for master-1",
			"Rollback:",
			"master-2:restoreVMSize",
			"master-2:restoreMachine",
			"master-2:restoreNodeLabels",
		)
	})
}

func TestNewResizeControlPlaneExecutionContext(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := newResizeControlPlaneExecutionContext(parent)
	cancelParent()
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatalf("execution context should outlive parent cancellation: %v", ctx.Err())
	default:
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("execution context should have a deadline")
	}

	expected := time.Duration(api.ControlPlaneNodeCount) * resizeControlPlanePerNodeExecutionTimeout
	remaining := time.Until(deadline)
	if remaining < expected-time.Second || remaining > expected {
		t.Fatalf("execution deadline remaining %s, want close to %s", remaining, expected)
	}
}

func TestNewResizeControlPlaneRollbackContext(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := newResizeControlPlaneRollbackContext(parent)
	cancelParent()
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatalf("rollback context should outlive parent cancellation: %v", ctx.Err())
	default:
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("rollback context should have a deadline")
	}

	expected := time.Duration(api.ControlPlaneNodeCount) * resizeControlPlanePerNodeRollbackTimeout
	remaining := time.Until(deadline)
	if remaining < expected-time.Second || remaining > expected {
		t.Fatalf("rollback deadline remaining %s, want close to %s", remaining, expected)
	}
}

func TestRecordOriginalVMSizeUsesAzureActualVM(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k := mock_adminactions.NewMockKubeActions(ctrl)
	a := mock_adminactions.NewMockAzureActions(ctrl)

	op := newResizeControlPlaneOperation(ctx, log, k, a, "Standard_D16s_v5", true, "test-cluster")
	machine := machineValidationData{
		size:              "Standard_D8s_v3",
		phase:             "Running",
		labelInstanceType: "Standard_D8s_v3",
	}

	gomock.InOrder(
		a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "master-0", mgmtcompute.InstanceView).
			Return(mgmtcompute.VirtualMachine{}, errors.New("azure get failed")),
	)

	_, err := op.captureNodeSnapshot(ctx, "master-0", machine)
	assertErrorContainsAll(t, err, "failed to capture Azure VM state for master-0")

	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	k = mock_adminactions.NewMockKubeActions(ctrl)
	a = mock_adminactions.NewMockAzureActions(ctrl)
	op = newResizeControlPlaneOperation(ctx, log, k, a, "Standard_D16s_v5", true, "test-cluster")

	gomock.InOrder(
		a.EXPECT().GetVirtualMachine(gomock.Any(), "test-cluster", "master-0", mgmtcompute.InstanceView).
			Return(virtualMachineWithSize("Standard_D4s_v3"), nil),
	)

	_, err = op.captureNodeSnapshot(ctx, "master-0", machine)
	assertErrorContainsAll(t, err, "actual Azure VM size Standard_D4s_v3", "Machine spec size Standard_D8s_v3")
}

func TestRollbackNodeRestoresMetadataWhenVMSizeIsAlreadyRestored(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()
	desiredSize := "Standard_D16s_v5"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k := mock_adminactions.NewMockKubeActions(ctrl)
	a := mock_adminactions.NewMockAzureActions(ctrl)
	op := newResizeControlPlaneOperation(ctx, log, k, a, "Standard_D16s_v5", true, "test-cluster")
	state := &controlPlaneNodeProgress{
		snapshot: controlPlaneNodeSnapshot{
			machineName:                  "master-0",
			originalVMSize:               "Standard_D8s_v3",
			originalMachineSize:          "Standard_D8s_v3",
			originalNodeInstanceType:     "Standard_D8s_v3",
			originalNodeBetaInstanceType: "Standard_D8s_v3",
			originallySchedulable:        true,
		},
		vmResized:                  true,
		machineUpdated:             true,
		nodeLabelsUpdated:          true,
		schedulabilityNeedsRestore: true,
	}

	gomock.InOrder(
		a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
		a.EXPECT().VMResize(gomock.Any(), "master-0", "Standard_D8s_v3").Return(nil),
		a.EXPECT().VMStartAndWait(gomock.Any(), "master-0").Return(errors.New("start failed after size restore")),
		k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-0").
			Return(machineJSON("master-0", desiredSize), nil),
		k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
		k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
			Return(nodeJSON("master-0", true), nil),
		k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
	)

	err := op.rollbackNode(ctx, state)
	assertErrorContainsAll(t, err,
		"restoring original VM size",
		"starting VM after restoring original size: start failed after size restore",
	)

	if state.schedulabilityNeedsRestore != true {
		t.Fatalf("schedulability should still need restore when node never became ready, got %v", state.schedulabilityNeedsRestore)
	}
}

func TestAdminReplyPreservesWrappedCloudError(t *testing.T) {
	recorder := httptest.NewRecorder()
	_, log := testlog.New()

	err := &resizeControlPlaneError{
		baseErr: api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "controlPlaneInventory", "inventory mismatch"),
	}

	adminReply(log, recorder, nil, nil, err)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusConflict)
	}

	var body map[string]map[string]any
	if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &body); decodeErr != nil {
		t.Fatalf("failed to decode response body: %v", decodeErr)
	}

	errorBody := body["error"]
	if errorBody["code"] != api.CloudErrorCodeRequestNotAllowed {
		t.Fatalf("error code = %v, want %s", errorBody["code"], api.CloudErrorCodeRequestNotAllowed)
	}
}
