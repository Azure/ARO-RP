package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/mock/gomock"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const executionContextCreationTolerance = time.Second

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

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()
		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to resize node master-0: resize: Azure resize error",
			"Steps:",
			"master-0:cordon",
			"master-0:drain",
			"master-0:stop",
			"master-0:resize failed",
			"master-0:start",
			"master-0:waitReady",
			"master-0:waitEtcd",
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

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()
		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to resize node master-1: drain: could not drain node after 3 retries: drain error",
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

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()
		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"Control plane node master-2 is not Ready",
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

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()
		err := resizeControlPlane(ctx, log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err,
			"failed to capture Azure VM state for master-1",
			"master-2:restoreVMSize",
			"master-2:restoreMachine",
			"master-2:restoreNodeLabels",
		)
	})
}

func TestNewResizeControlPlaneExecutionContext(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	start := time.Now()
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

	expected := time.Duration(api.ControlPlaneNodeCount) * resizeControlPlanePerNodeTimeout
	actual := deadline.Sub(start)
	if actual < expected || actual > expected+executionContextCreationTolerance {
		t.Fatalf("execution deadline duration %s, want in [%s, %s]", actual, expected, expected+executionContextCreationTolerance)
	}
}

func TestRecordOriginalVMSizeUsesAzureActualVM(t *testing.T) {
	ctx := context.Background()
	_, log := testlog.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k := mock_adminactions.NewMockKubeActions(ctrl)
	a := mock_adminactions.NewMockAzureActions(ctrl)

	op := newResizeControlPlaneOperation(log, k, a, "Standard_D16s_v5", true, "test-cluster")
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
	op = newResizeControlPlaneOperation(log, k, a, "Standard_D16s_v5", true, "test-cluster")

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
	op := newResizeControlPlaneOperation(log, k, a, desiredSize, true, "test-cluster")
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
		a.EXPECT().VMStartAndWait(gomock.Any(), "master-0").Return(errors.New("start failed after size restore")).Times(azureOperationMaxAttempts),
		k.EXPECT().KubeGet(gomock.Any(), "Machine.machine.openshift.io", machineNamespace, "master-0").
			Return(machineJSON("master-0", desiredSize), nil),
		k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
		k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
			Return(nodeJSON("master-0", true), nil),
		k.EXPECT().KubeCreateOrUpdate(gomock.Any(), gomock.Any()).Return(nil),
	)

	err := op.rollbackNode(ctx, state)
	assertErrorContainsAll(t, err,
		"starting VM after restoring original size",
		"start failed after size restore",
	)

	if state.schedulabilityNeedsRestore != true {
		t.Fatalf("schedulability should still need restore when node never became ready, got %v", state.schedulabilityNeedsRestore)
	}
}

func TestRollbackAllFailsFastWhenEtcdUnhealthyBetweenNodes(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	k := mock_adminactions.NewMockKubeActions(ctrl)
	a := mock_adminactions.NewMockAzureActions(ctrl)

	op := &resizeControlPlaneOperation{
		k: k,
		a: a,
		nodes: []*controlPlaneNodeProgress{
			{snapshot: controlPlaneNodeSnapshot{machineName: "master-0"}},
			{snapshot: controlPlaneNodeSnapshot{machineName: "master-1"}},
		},
	}

	k.EXPECT().
		KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
		Return(nil, errors.New("api server unavailable"))

	err := op.rollbackAll(ctx)
	assertErrorContainsAll(t, err, "etcd unhealthy before rollback of master-0", "api server unavailable")
}

func TestAdminReplyPreservesWrappedCloudError(t *testing.T) {
	recorder := httptest.NewRecorder()
	_, log := testlog.New()

	err := &resizeControlPlaneError{
		baseErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "controlPlaneInventory", "inventory mismatch"),
		steps: []string{
			"master-0:stop (10ms)",
			"master-0:resize failed (10ms): Azure resize error",
			"master-0:start (10ms)",
		},
	}

	adminReply(log, recorder, nil, nil, normalizeResizeControlPlaneErrorForAdminReply(err))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var body map[string]map[string]any
	if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &body); decodeErr != nil {
		t.Fatalf("failed to decode response body: %v", decodeErr)
	}

	errorBody := body["error"]
	if errorBody["code"] != api.CloudErrorCodeInvalidParameter {
		t.Fatalf("error code = %v, want %s", errorBody["code"], api.CloudErrorCodeInvalidParameter)
	}
	if errorBody["target"] != "controlPlaneInventory" {
		t.Fatalf("error target = %v, want %s", errorBody["target"], "controlPlaneInventory")
	}

	message, ok := errorBody["message"].(string)
	if !ok {
		t.Fatalf("error message has unexpected type %T", errorBody["message"])
	}
	for _, expected := range []string{
		"inventory mismatch",
		"Steps:",
		"master-0:stop",
		"master-0:resize failed",
		"master-0:start",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error message %q does not contain %q", message, expected)
		}
	}
}

func TestAdminReplyFallsBackTo500WhenNoCloudError(t *testing.T) {
	recorder := httptest.NewRecorder()
	_, log := testlog.New()

	err := &resizeControlPlaneError{
		baseErr: fmt.Errorf("failed to resize node master-0: resize: %w", errors.New("Azure resize error")),
		steps: []string{
			"master-0:stop (10ms)",
			"master-0:resize failed (10ms): Azure resize error",
			"master-0:start (10ms)",
		},
	}

	adminReply(log, recorder, nil, nil, normalizeResizeControlPlaneErrorForAdminReply(err))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var body map[string]map[string]any
	if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &body); decodeErr != nil {
		t.Fatalf("failed to decode response body: %v", decodeErr)
	}

	errorBody := body["error"]
	if errorBody["code"] != api.CloudErrorCodeInternalServerError {
		t.Fatalf("error code = %v, want %s", errorBody["code"], api.CloudErrorCodeInternalServerError)
	}

	message, ok := errorBody["message"].(string)
	if !ok {
		t.Fatalf("error message has unexpected type %T", errorBody["message"])
	}
	for _, expected := range []string{
		"Azure resize error",
		"Steps:",
		"master-0:stop",
		"master-0:resize failed",
		"master-0:start",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error message %q does not contain %q", message, expected)
		}
	}
}

func TestAdminReplyPreservesWrappedCloudErrorWithoutBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	_, log := testlog.New()

	err := &resizeControlPlaneError{
		baseErr: &api.CloudError{StatusCode: http.StatusBadGateway},
		steps: []string{
			"master-0:stop (10ms)",
			"master-0:resize failed (10ms): Azure resize error",
		},
	}

	adminReply(log, recorder, nil, nil, normalizeResizeControlPlaneErrorForAdminReply(err))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadGateway)
	}

	var body map[string]map[string]any
	if decodeErr := json.Unmarshal(recorder.Body.Bytes(), &body); decodeErr != nil {
		t.Fatalf("failed to decode response body: %v", decodeErr)
	}

	errorBody := body["error"]
	if errorBody["code"] != api.CloudErrorCodeInternalServerError {
		t.Fatalf("error code = %v, want %s", errorBody["code"], api.CloudErrorCodeInternalServerError)
	}

	message, ok := errorBody["message"].(string)
	if !ok {
		t.Fatalf("error message has unexpected type %T", errorBody["message"])
	}
	for _, expected := range []string{
		"502",
		"Steps:",
		"master-0:stop",
		"master-0:resize failed",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("error message %q does not contain %q", message, expected)
		}
	}
}

func TestRetryAzureOperation(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		calls := 0
		err := retryAzureOperation(context.Background(), "test op", func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 call, got %d", calls)
		}
	})

	t.Run("succeeds on second attempt", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			calls := 0
			err := retryAzureOperation(context.Background(), "test op", func() error {
				calls++
				if calls == 1 {
					return errors.New("transient")
				}
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if calls != 2 {
				t.Fatalf("expected 2 calls, got %d", calls)
			}
		})
	})

	t.Run("fails after max attempts", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			calls := 0
			err := retryAzureOperation(context.Background(), "test op", func() error {
				calls++
				return errors.New("persistent")
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if calls != azureOperationMaxAttempts {
				t.Fatalf("expected %d calls, got %d", azureOperationMaxAttempts, calls)
			}
			assertErrorContainsAll(t, err, "could not complete test op", "persistent")
		})
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := retryAzureOperation(ctx, "test op", func() error {
			return errors.New("will retry")
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("allows injected retry delay policy", func(t *testing.T) {
		policy := retryAzureOperationPolicy{
			maxAttempts: 2,
			retryDelay:  0,
		}
		calls := 0
		err := retryAzureOperationWithPolicy(context.Background(), "test op", policy, func() error {
			calls++
			return errors.New("persistent")
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if calls != policy.maxAttempts {
			t.Fatalf("expected %d calls, got %d", policy.maxAttempts, calls)
		}
		assertErrorContainsAll(t, err, "could not complete test op", "persistent")
	})
}
