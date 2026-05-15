package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
)

const (
	// The forward path historically used a 45 minute budget. Keep that as a
	// per-master execution budget instead of a fixed whole-operation cap.
	resizeControlPlanePerNodeExecutionTimeout = 45 * time.Minute
	// Rollback needs a larger per-master budget because a late failure can
	// require stop -> resize back -> start -> waitReady before metadata restore.
	resizeControlPlanePerNodeRollbackTimeout = time.Hour

	azureOperationMaxAttempts = 3
	azureOperationRetryDelay  = 5 * time.Second

	etcdHealthPollTimeout  = 10 * time.Minute
	etcdHealthPollInterval = 10 * time.Second
)

type resizeStep string

const (
	resizeStepCordon                resizeStep = "cordon"
	resizeStepDrain                 resizeStep = "drain"
	resizeStepStop                  resizeStep = "stop"
	resizeStepResize                resizeStep = "resize"
	resizeStepStart                 resizeStep = "start"
	resizeStepWaitReady             resizeStep = "waitReady"
	resizeStepUncordon              resizeStep = "uncordon"
	resizeStepUpdateMachine         resizeStep = "updateMachine"
	resizeStepUpdateNodeLabels      resizeStep = "updateNodeLabels"
	resizeStepRestoreVMSize         resizeStep = "restoreVMSize"
	resizeStepRestoreMachine        resizeStep = "restoreMachine"
	resizeStepRestoreNodeLabels     resizeStep = "restoreNodeLabels"
	resizeStepRestoreSchedulability resizeStep = "restoreSchedulability"
	resizeStepWaitEtcd              resizeStep = "waitEtcdHealthy"
)

type resizeStepRecord struct {
	nodeName       string
	step           resizeStep
	rollback       bool
	duration       time.Duration
	err            error
	originalVMSize string
}

type resizeStepError struct {
	nodeName string
	step     resizeStep
	err      error
}

func (e *resizeStepError) Error() string {
	return e.err.Error()
}

func (e *resizeStepError) Unwrap() error {
	return e.err
}

type resizeControlPlaneError struct {
	baseErr     error
	forward     []resizeStepRecord
	rollback    []resizeStepRecord
	rollbackErr error
}

func (e *resizeControlPlaneError) Error() string {
	var b strings.Builder
	b.WriteString(e.baseErr.Error())

	if len(e.forward) > 0 {
		b.WriteString(". Steps taken: ")
		b.WriteString(formatResizeStepRecords(e.forward))
	}
	if len(e.rollback) > 0 {
		b.WriteString(". Rollback: ")
		b.WriteString(formatResizeStepRecords(e.rollback))
	}
	if e.rollbackErr != nil {
		b.WriteString(". Rollback errors: ")
		b.WriteString(e.rollbackErr.Error())
	}

	return b.String()
}

func (e *resizeControlPlaneError) Unwrap() error {
	return e.baseErr
}

type controlPlaneNodeSnapshot struct {
	machineName                  string
	originalVMSize               string
	originalMachineSize          string
	originalNodeInstanceType     string
	originalNodeBetaInstanceType string
	originallySchedulable        bool
}

type controlPlaneNodeProgress struct {
	snapshot                   controlPlaneNodeSnapshot
	vmStopped                  bool
	vmResized                  bool
	machineUpdated             bool
	nodeLabelsUpdated          bool
	schedulabilityNeedsRestore bool
}

type resizeControlPlaneOperation struct {
	log                      *logrus.Entry
	k                        adminactions.KubeActions
	a                        adminactions.AzureActions
	desiredVMSize            string
	deallocateVM             bool
	clusterResourceGroupName string
	forward                  []resizeStepRecord
	rollback                 []resizeStepRecord
	nodes                    []*controlPlaneNodeProgress
}

// newResizeControlPlaneExecutionContext creates a context decoupled from the
// HTTP request lifecycle. The resize operation must complete even if the client
// disconnects or the ARM load balancer times out (typical: 4-10 min).
//
// Trade-offs:
//   - The HTTP response may be written to a dead connection; the SRE must check
//     logs or cluster state to confirm the outcome.
//   - If the RP pod is recycled (graceful shutdown), this context will NOT be
//     cancelled. The goroutine continues until SIGKILL after the termination
//     grace period. This can leave the cluster in a partially resized state
//     requiring manual recovery via the Azure portal.
//   - The CosmosDB provisioning state lock (ProvisioningStateAdminUpdating)
//     ensures a concurrent resize attempt will be rejected with 409, but the
//     lock may not be released if the pod is killed mid-operation.
//
// This endpoint is restricted to on-call SREs via Geneva Actions
// (ClientPlatformServiceOperator claim).
func newResizeControlPlaneExecutionContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(parent),
		time.Duration(api.ControlPlaneNodeCount)*resizeControlPlanePerNodeExecutionTimeout,
	)
}

// newResizeControlPlaneRollbackContext creates a context for rollback operations.
// Like the execution context, it uses WithoutCancel to ensure rollback completes
// even if the original request is cancelled. The rollback budget is larger
// because restoring the original VM size requires stop -> resize -> start ->
// waitReady before metadata can be restored.
func newResizeControlPlaneRollbackContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(parent),
		time.Duration(api.ControlPlaneNodeCount)*resizeControlPlanePerNodeRollbackTimeout,
	)
}

func newResizeControlPlaneOperation(
	_ context.Context,
	log *logrus.Entry,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
	desiredVMSize string,
	deallocateVM bool,
	clusterResourceGroupName string,
) *resizeControlPlaneOperation {
	return &resizeControlPlaneOperation{
		log:                      log,
		k:                        k,
		a:                        a,
		desiredVMSize:            desiredVMSize,
		deallocateVM:             deallocateVM,
		clusterResourceGroupName: clusterResourceGroupName,
	}
}

func (o *resizeControlPlaneOperation) captureNodeSnapshot(ctx context.Context, machineName string, machine machineValidationData) (controlPlaneNodeSnapshot, error) {
	if machine.phase != "Running" {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed,
			"",
			fmt.Sprintf("control plane machine %s is not Running (phase=%s)", machineName, machine.phase),
		)
	}
	if machine.labelInstanceType == "" || machine.labelInstanceType != machine.size {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed,
			"",
			fmt.Sprintf("control plane machine %s has mismatched Machine metadata: label instance-type %q, spec size %q", machineName, machine.labelInstanceType, machine.size),
		)
	}

	vm, err := o.a.GetVirtualMachine(ctx, o.clusterResourceGroupName, machineName, mgmtcompute.InstanceView)
	if err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture Azure VM state for %s: %w", machineName, err)
	}
	if vm.VirtualMachineProperties == nil || vm.HardwareProfile == nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture Azure VM state for %s: HardwareProfile missing", machineName)
	}

	actualVMSize := string(vm.HardwareProfile.VMSize)
	if !strings.EqualFold(actualVMSize, machine.size) {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed,
			"",
			fmt.Sprintf("actual Azure VM size %s does not match Machine spec size %s for %s", actualVMSize, machine.size, machineName),
		)
	}

	rawNode, err := o.k.KubeGet(ctx, "Node", "", machineName)
	if err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture node state for %s: %w", machineName, err)
	}

	var node corev1.Node
	if err := json.Unmarshal(rawNode, &node); err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to parse node state for %s: %w", machineName, err)
	}

	labels := node.GetLabels()
	nodeInstanceType := labels[nodeLabelInstanceType]
	betaInstanceType := labels[nodeLabelBetaInstanceType]
	if nodeInstanceType != betaInstanceType {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed,
			"",
			fmt.Sprintf("node %s has inconsistent instance type labels: %s=%q, %s=%q", machineName, nodeLabelInstanceType, nodeInstanceType, nodeLabelBetaInstanceType, betaInstanceType),
		)
	}
	if !strings.EqualFold(nodeInstanceType, machine.size) {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed,
			"",
			fmt.Sprintf("node %s instance type labels do not match Machine spec size %s", machineName, machine.size),
		)
	}

	snapshot := controlPlaneNodeSnapshot{
		machineName:                  machineName,
		originalVMSize:               actualVMSize,
		originalMachineSize:          machine.size,
		originalNodeInstanceType:     nodeInstanceType,
		originalNodeBetaInstanceType: betaInstanceType,
		originallySchedulable:        !node.Spec.Unschedulable,
	}

	o.log.WithFields(logrus.Fields{
		"node":           machineName,
		"originalVMSize": actualVMSize,
		"targetVMSize":   o.desiredVMSize,
	}).Info("captured node snapshot for resize operation")

	return snapshot, nil
}

func (o *resizeControlPlaneOperation) runStep(ctx context.Context, nodeName string, step resizeStep, rollback bool, wrapPrefix string, fn func(context.Context) error) error {
	return o.runStepWithSnapshot(ctx, nodeName, "", step, rollback, wrapPrefix, fn)
}

func (o *resizeControlPlaneOperation) runStepWithSnapshot(ctx context.Context, nodeName, originalVMSize string, step resizeStep, rollback bool, wrapPrefix string, fn func(context.Context) error) error {
	start := time.Now()
	err := fn(ctx)
	record := resizeStepRecord{
		nodeName:       nodeName,
		step:           step,
		rollback:       rollback,
		duration:       time.Since(start),
		err:            err,
		originalVMSize: originalVMSize,
	}
	if rollback {
		o.rollback = append(o.rollback, record)
	} else {
		o.forward = append(o.forward, record)
	}
	if err != nil {
		return &resizeStepError{
			nodeName: nodeName,
			step:     step,
			err:      fmt.Errorf("%s: %w", wrapPrefix, err),
		}
	}

	return nil
}

func (o *resizeControlPlaneOperation) resizeNode(ctx context.Context, state *controlPlaneNodeProgress) error {
	nodeName := state.snapshot.machineName

	if state.snapshot.originallySchedulable {
		if err := o.runStep(ctx, nodeName, resizeStepCordon, false, "cordoning node", func(ctx context.Context) error {
			return cordonNode(ctx, o.k, nodeName)
		}); err != nil {
			return err
		}
		state.schedulabilityNeedsRestore = true
	}

	if err := o.runStep(ctx, nodeName, resizeStepDrain, false, "draining node", func(ctx context.Context) error {
		return o.k.DrainNodeWithRetries(ctx, nodeName)
	}); err != nil {
		return err
	}

	if err := o.runStep(ctx, nodeName, resizeStepStop, false, "stopping VM", func(ctx context.Context) error {
		return o.a.VMStopAndWait(ctx, nodeName, o.deallocateVM)
	}); err != nil {
		return err
	}
	state.vmStopped = true

	if err := o.runStep(ctx, nodeName, resizeStepResize, false, "resizing VM", func(ctx context.Context) error {
		return o.a.VMResize(ctx, nodeName, o.desiredVMSize)
	}); err != nil {
		return err
	}
	state.vmResized = true

	if err := o.runStep(ctx, nodeName, resizeStepStart, false, "starting VM", func(ctx context.Context) error {
		return o.a.VMStartAndWait(ctx, nodeName)
	}); err != nil {
		return err
	}

	if err := o.runStep(ctx, nodeName, resizeStepWaitReady, false, "waiting for node ready", func(ctx context.Context) error {
		return waitForNodeReady(ctx, o.log, o.k, nodeName)
	}); err != nil {
		return err
	}
	state.vmStopped = false

	if err := o.runStep(ctx, nodeName, resizeStepWaitEtcd, false, "waiting for etcd healthy", func(ctx context.Context) error {
		return waitForEtcdHealthy(ctx, o.log, o.k)
	}); err != nil {
		return err
	}

	if state.snapshot.originallySchedulable {
		if err := o.runStep(ctx, nodeName, resizeStepUncordon, false, "uncordoning node", func(ctx context.Context) error {
			return uncordonNode(ctx, o.k, nodeName)
		}); err != nil {
			return err
		}
		state.schedulabilityNeedsRestore = false
	}

	if err := o.runStep(ctx, nodeName, resizeStepUpdateMachine, false, "updating Machine object", func(ctx context.Context) error {
		return updateMachineVMSize(ctx, o.k, nodeName, o.desiredVMSize)
	}); err != nil {
		return err
	}
	state.machineUpdated = true

	if err := o.runStep(ctx, nodeName, resizeStepUpdateNodeLabels, false, "updating Node labels", func(ctx context.Context) error {
		return updateNodeInstanceTypeLabels(ctx, o.k, nodeName, o.desiredVMSize)
	}); err != nil {
		return err
	}
	state.nodeLabelsUpdated = true

	return nil
}

func waitForEtcdHealthy(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions) error {
	ctx, cancel := context.WithTimeout(ctx, etcdHealthPollTimeout)
	defer cancel()

	return wait.PollImmediateUntilWithContext(ctx, etcdHealthPollInterval, func(ctx context.Context) (bool, error) {
		if err := validateEtcdHealth(ctx, k); err != nil {
			log.Infof("Waiting for etcd to become healthy: %v", err)
			return false, nil
		}
		return true, nil
	})
}

func ensureControlPlaneAndEtcdHealthy(ctx context.Context, k adminactions.KubeActions, nodeNames []string) error {
	if err := ensureControlPlaneNodesReadyAndSchedulable(ctx, k, nodeNames); err != nil {
		return err
	}
	return validateEtcdHealth(ctx, k)
}

func retryAzureOperation(ctx context.Context, operationDesc string, fn func() error) error {
	var lastErr error
	for attempt := range azureOperationMaxAttempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == azureOperationMaxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(azureOperationRetryDelay):
		}
	}
	return fmt.Errorf("could not complete %s after %d attempts: %w", operationDesc, azureOperationMaxAttempts, lastErr)
}

func (o *resizeControlPlaneOperation) rollbackNode(ctx context.Context, state *controlPlaneNodeProgress) error {
	nodeName := state.snapshot.machineName
	var rollbackErrs []error
	vmSizeRestored := !state.vmResized
	nodeReadyForSchedRestore := !state.vmStopped && !state.vmResized

	if state.vmResized {
		err := o.runStep(ctx, nodeName, resizeStepRestoreVMSize, true, "restoring original VM size", func(ctx context.Context) error {
			// Rollback always uses a deallocated stop so restoring the original SKU
			// takes the most conservative Azure path, even if the forward request
			// preferred a non-deallocated stop.
			if err := retryAzureOperation(ctx, "stop VM for rollback", func() error {
				return o.a.VMStopAndWait(ctx, nodeName, true)
			}); err != nil {
				return fmt.Errorf("stopping VM before restoring original size: %w", err)
			}
			if err := retryAzureOperation(ctx, "resize VM for rollback", func() error {
				return o.a.VMResize(ctx, nodeName, state.snapshot.originalVMSize)
			}); err != nil {
				return fmt.Errorf("restoring VM size to %s: %w", state.snapshot.originalVMSize, err)
			}
			vmSizeRestored = true
			state.vmResized = false
			o.log.Infof("VM size for %s successfully restored to %s; continuing with VM start", nodeName, state.snapshot.originalVMSize)
			if err := retryAzureOperation(ctx, "start VM after rollback", func() error {
				return o.a.VMStartAndWait(ctx, nodeName)
			}); err != nil {
				return fmt.Errorf("starting VM after restoring original size: %w", err)
			}
			if err := waitForNodeReady(ctx, o.log, o.k, nodeName); err != nil {
				return fmt.Errorf("waiting for node ready after restoring original size: %w", err)
			}
			if err := waitForEtcdHealthy(ctx, o.log, o.k); err != nil {
				return fmt.Errorf("waiting for etcd healthy after restoring original size: %w", err)
			}
			nodeReadyForSchedRestore = true
			return nil
		})
		if err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.vmStopped = false
		}
	} else if state.vmStopped {
		nodeReadyForSchedRestore = false
		if err := o.runStep(ctx, nodeName, resizeStepStart, true, "starting VM during rollback", func(ctx context.Context) error {
			return retryAzureOperation(ctx, "start VM during rollback", func() error {
				return o.a.VMStartAndWait(ctx, nodeName)
			})
		}); err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else if err := o.runStep(ctx, nodeName, resizeStepWaitReady, true, "waiting for node ready during rollback", func(ctx context.Context) error {
			return waitForNodeReady(ctx, o.log, o.k, nodeName)
		}); err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.vmStopped = false
			nodeReadyForSchedRestore = true
		}
	}

	if state.machineUpdated && vmSizeRestored {
		if err := o.runStep(ctx, nodeName, resizeStepRestoreMachine, true, "restoring Machine object", func(ctx context.Context) error {
			return updateMachineVMSize(ctx, o.k, nodeName, state.snapshot.originalMachineSize)
		}); err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.machineUpdated = false
		}
	}

	if state.nodeLabelsUpdated && vmSizeRestored {
		if err := o.runStep(ctx, nodeName, resizeStepRestoreNodeLabels, true, "restoring Node labels", func(ctx context.Context) error {
			return restoreNodeInstanceTypeLabels(ctx, o.k, nodeName, state.snapshot.originalNodeInstanceType, state.snapshot.originalNodeBetaInstanceType)
		}); err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.nodeLabelsUpdated = false
		}
	}

	if state.schedulabilityNeedsRestore && nodeReadyForSchedRestore {
		var restoreFn func(context.Context) error
		if state.snapshot.originallySchedulable {
			restoreFn = func(ctx context.Context) error { return uncordonNode(ctx, o.k, nodeName) }
		} else {
			restoreFn = func(ctx context.Context) error { return cordonNode(ctx, o.k, nodeName) }
		}

		if err := o.runStep(ctx, nodeName, resizeStepRestoreSchedulability, true, "restoring node schedulability", restoreFn); err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.schedulabilityNeedsRestore = false
		}
	}

	return errors.Join(rollbackErrs...)
}

func (o *resizeControlPlaneOperation) rollbackAll(ctx context.Context) error {
	var errs []error
	for i := len(o.nodes) - 1; i >= 0; i-- {
		if i < len(o.nodes)-1 {
			if err := validateEtcdHealth(ctx, o.k); err != nil {
				o.log.Warnf("etcd not fully healthy before rollback of %s: %v", o.nodes[i].snapshot.machineName, err)
			}
		}
		if err := o.rollbackNode(ctx, o.nodes[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func formatResizeStepRecords(records []resizeStepRecord) string {
	parts := make([]string, 0, len(records))
	for _, record := range records {
		duration := record.duration.Truncate(time.Millisecond)
		if duration <= 0 {
			duration = record.duration
		}
		if duration <= 0 {
			duration = time.Millisecond
		}

		nodeID := record.nodeName
		if record.originalVMSize != "" {
			nodeID = fmt.Sprintf("%s[%s]", record.nodeName, record.originalVMSize)
		}

		entry := fmt.Sprintf("%s:%s (%s)", nodeID, record.step, duration)
		if record.err != nil {
			entry = fmt.Sprintf("%s failed (%s): %v", fmt.Sprintf("%s:%s", nodeID, record.step), duration, record.err)
		}
		parts = append(parts, entry)
	}

	return strings.Join(parts, ", ")
}
