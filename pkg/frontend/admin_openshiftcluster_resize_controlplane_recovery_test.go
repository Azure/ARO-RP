package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

const executionContextCreationTolerance = time.Second

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

func TestRunStepWrapsStepErrors(t *testing.T) {
	_, log := testlog.LogForTesting(t)

	op := &resizeControlPlaneOperation{log: log}

	err := op.runStep(t.Context(), "resize", steps.Action(func(context.Context) error {
		return errors.New("boom")
	}))

	assertErrorContainsAll(t, err, "resize: boom")
}

func TestRunStepPreservesOriginalActionErrors(t *testing.T) {
	_, log := testlog.LogForTesting(t)

	op := &resizeControlPlaneOperation{log: log}

	err := op.runStep(t.Context(), "start", steps.Action(func(context.Context) error {
		return errors.New("AADSTS700016 original Azure failure")
	}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) {
		t.Fatalf("runStep should not reclassify action errors into CloudError, got %#v", cloudErr)
	}
	assertErrorContainsAll(t, err, "start: AADSTS700016 original Azure failure")
}

func TestWaitForEtcdHealthyBoundsHealthCheckByEtcdTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, log := testlog.LogForTesting(t)

		ctrl := gomock.NewController(t)

		k := mock_adminactions.NewMockKubeActions(ctrl)

		start := time.Now()
		k.EXPECT().
			KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			DoAndReturn(func(ctx context.Context, resource, namespace, name string) ([]byte, error) {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("expected waitForEtcdHealthy to bound each etcd health check with a deadline")
				}

				got := deadline.Sub(start)
				if got < etcdHealthPollTimeout || got > etcdHealthPollTimeout+time.Second {
					t.Fatalf("etcd health-check deadline = %s, want in [%s, %s]", got, etcdHealthPollTimeout, etcdHealthPollTimeout+time.Second)
				}

				<-ctx.Done()
				return nil, ctx.Err()
			}).
			Times(1)

		err := waitForEtcdHealthy(t.Context(), log, k)
		assertErrorContainsAll(t, err,
			"failed to retrieve etcd ClusterOperator",
			context.DeadlineExceeded.Error(),
		)
	})
}

func TestResizeControlPlaneFailsWithoutRollback(t *testing.T) {
	_, log := testlog.LogForTesting(t)

	running := "Running"
	desiredSize := "Standard_D16s_v5"
	clusterResourceGroupName := "test-cluster"

	t.Run("returns at the failing step on a single node", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), machineGroupKind, machineNamespace).Return(
			masterMachineListJSON(masterMachine("master-0", "Standard_D8s_v3", running)), nil)

		gomock.InOrder(
			// pre-loop health gate
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").
				Return(nodeJSON("master-0", true), nil),
			k.EXPECT().CordonNode(gomock.Any(), "master-0", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-0").Return(nil),
			a.EXPECT().VMStopAndWait(gomock.Any(), "master-0", true).Return(nil),
			a.EXPECT().VMResize(gomock.Any(), "master-0", desiredSize).Return(errors.New("Azure resize error")),
		)

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()

		err := resizeControlPlane(t.Context(), log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err, "failed to resize node master-0: resize: Azure resize error")
	})

	t.Run("leaves an already-resized node untouched when a later node fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		k := mock_adminactions.NewMockKubeActions(ctrl)
		a := mock_adminactions.NewMockAzureActions(ctrl)

		k.EXPECT().KubeList(gomock.Any(), machineGroupKind, machineNamespace).Return(
			masterMachineListJSON(
				masterMachine("master-0", desiredSize, running),
				masterMachine("master-1", "Standard_D8s_v3", running),
				masterMachine("master-2", "Standard_D8s_v3", running),
			), nil)

		gomock.InOrder(
			// iteration for master-2: health gate
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			// master-2 resizes successfully
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

			// iteration for master-1: health gate
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-2").Return(nodeJSON("master-2", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-1").Return(nodeJSON("master-1", true), nil),
			k.EXPECT().KubeGet(gomock.Any(), "Node", "", "master-0").Return(nodeJSON("master-0", true), nil),

			// master-1 resize fails at drain, no rollback of master-2
			k.EXPECT().CordonNode(gomock.Any(), "master-1", true).Return(nil),
			k.EXPECT().DrainNodeWithRetries(gomock.Any(), "master-1").Return(errors.New("could not drain node after 3 retries: drain error")),
		)

		k.EXPECT().KubeGet(gomock.Any(), "ClusterOperator.config.openshift.io", "", "etcd").
			Return(healthyEtcdJSON(), nil).AnyTimes()

		err := resizeControlPlane(t.Context(), log, k, a, desiredSize, true, clusterResourceGroupName)
		assertErrorContainsAll(t, err, "failed to resize node master-1: drain: could not drain node after 3 retries: drain error")
	})
}

func TestNewResizeControlPlaneExecutionContext(t *testing.T) {
	parent, cancelParent := context.WithCancel(t.Context())
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
