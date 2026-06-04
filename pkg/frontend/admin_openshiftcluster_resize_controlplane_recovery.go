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
	resizeControlPlanePerNodeTimeout = 45 * time.Minute

	azureOperationMaxAttempts = 3
	azureOperationRetryDelay  = 5 * time.Second

	etcdHealthPollTimeout  = 10 * time.Minute
	etcdHealthPollInterval = 10 * time.Second
)

// retryAzureOperationPolicy defines the maximum number of attempts and the delay between attempts for retrying Azure operations.
// Production keeps current behavior and tests can override delay.
type retryAzureOperationPolicy struct {
	maxAttempts int
	retryDelay  time.Duration
}

var defaultRetryAzureOperationPolicy = retryAzureOperationPolicy{
	maxAttempts: azureOperationMaxAttempts,
	retryDelay:  azureOperationRetryDelay,
}

type resizeControlPlaneError struct {
	baseErr     error
	steps       []string
	rollbackErr error
}

func (e *resizeControlPlaneError) Error() string {
	var b strings.Builder
	b.WriteString(e.baseErr.Error())

	if len(e.steps) > 0 {
		b.WriteString(". Steps: ")
		b.WriteString(strings.Join(e.steps, ", "))
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
	steps                    []string
	nodes                    []*controlPlaneNodeProgress
}

// newResizeControlPlaneExecutionContext creates a context decoupled from the
// HTTP request lifecycle. The resize operation (~45 min for 3 nodes) must
// complete even if the HTTP connection drops — the Geneva Actions client sets
// Timeout = InfiniteTimeSpan, but intermediary load balancers between the ACIS
// host and the RP VMSS can still close long-lived connections. ARM is not in
// this path; Geneva Actions calls the admin API directly via client certificate.
//
// Parallel invocation is unlikely — this is restricted to on-call SREs via
// Geneva Actions (ClientPlatformServiceOperator claim) and requires manual
// parameter entry. The real risk is not parallel calls but RP pod restart
// (OOM, node eviction, rolling update) or network partition during the
// operation, which would cancel the HTTP context and leave nodes mid-resize.
// Context detachment ensures the resize runs to completion or rolls back.
func newResizeControlPlaneExecutionContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(parent),
		time.Duration(api.ControlPlaneNodeCount)*resizeControlPlanePerNodeTimeout,
	)
}

func newResizeControlPlaneOperation(
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

func (o *resizeControlPlaneOperation) recordStep(node, step string, d time.Duration, err error) {
	if err != nil {
		o.steps = append(o.steps, fmt.Sprintf("%s:%s failed (%s): %v", node, step, d.Truncate(time.Millisecond), err))
	} else {
		o.steps = append(o.steps, fmt.Sprintf("%s:%s (%s)", node, step, d.Truncate(time.Millisecond)))
	}
}

func (o *resizeControlPlaneOperation) captureNodeSnapshot(ctx context.Context, machineName string, machine machineValidationData) (controlPlaneNodeSnapshot, error) {
	if machine.phase != "Running" {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("control plane machine %s is not Running (phase=%s)", machineName, machine.phase),
		)
	}
	if machine.labelInstanceType == "" || machine.labelInstanceType != machine.size {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
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
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
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
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
			"",
			fmt.Sprintf("node %s has inconsistent instance type labels: %s=%q, %s=%q", machineName, nodeLabelInstanceType, nodeInstanceType, nodeLabelBetaInstanceType, betaInstanceType),
		)
	}
	if !strings.EqualFold(nodeInstanceType, machine.size) {
		return controlPlaneNodeSnapshot{}, api.NewCloudError(
			http.StatusBadRequest,
			api.CloudErrorCodeInvalidParameter,
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

func (o *resizeControlPlaneOperation) resizeNode(ctx context.Context, state *controlPlaneNodeProgress) error {
	nodeName := state.snapshot.machineName

	run := func(step string, fn func() error) error {
		start := time.Now()
		err := fn()
		o.recordStep(nodeName, step, time.Since(start), err)
		if err != nil {
			return fmt.Errorf("%s: %w", step, err)
		}
		return nil
	}

	if state.snapshot.originallySchedulable {
		if err := run("cordon", func() error { return cordonNode(ctx, o.k, nodeName) }); err != nil {
			return err
		}
		state.schedulabilityNeedsRestore = true
	}

	if err := run("drain", func() error { return o.k.DrainNodeWithRetries(ctx, nodeName) }); err != nil {
		return err
	}

	if err := run("stop", func() error { return o.a.VMStopAndWait(ctx, nodeName, o.deallocateVM) }); err != nil {
		return err
	}
	state.vmStopped = true

	if err := run("resize", func() error { return o.a.VMResize(ctx, nodeName, o.desiredVMSize) }); err != nil {
		return err
	}
	state.vmResized = true

	if err := run("start", func() error { return o.a.VMStartAndWait(ctx, nodeName) }); err != nil {
		return err
	}

	if err := run("waitReady", func() error { return waitForNodeReady(ctx, o.log, o.k, nodeName) }); err != nil {
		return err
	}
	state.vmStopped = false

	if err := run("waitEtcd", func() error { return waitForEtcdHealthy(ctx, o.log, o.k) }); err != nil {
		return err
	}

	if state.snapshot.originallySchedulable {
		if err := run("uncordon", func() error { return uncordonNode(ctx, o.k, nodeName) }); err != nil {
			return err
		}
		state.schedulabilityNeedsRestore = false
	}

	if err := run("updateMachine", func() error { return updateMachineVMSize(ctx, o.k, nodeName, o.desiredVMSize) }); err != nil {
		return err
	}
	state.machineUpdated = true

	if err := run("updateNodeLabels", func() error { return updateNodeInstanceTypeLabels(ctx, o.k, nodeName, o.desiredVMSize) }); err != nil {
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
			var cloudErr *api.CloudError
			if errors.As(err, &cloudErr) && cloudErr.StatusCode == http.StatusConflict {
				log.Infof("Waiting for etcd to become healthy: %v", err)
				return false, nil
			}
			return false, err
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
	return retryAzureOperationWithPolicy(ctx, operationDesc, defaultRetryAzureOperationPolicy, fn)
}

func retryAzureOperationWithPolicy(
	ctx context.Context,
	operationDesc string,
	policy retryAzureOperationPolicy,
	fn func() error,
) error {
	if policy.maxAttempts <= 0 {
		return fmt.Errorf("could not complete %s: invalid retry policy max attempts %d", operationDesc, policy.maxAttempts)
	}

	var lastErr error
	for attempt := range policy.maxAttempts {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == policy.maxAttempts-1 {
			break
		}
		if policy.retryDelay <= 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(policy.retryDelay):
		}
	}
	return fmt.Errorf("could not complete %s after %d attempts: %w", operationDesc, policy.maxAttempts, lastErr)
}

func (o *resizeControlPlaneOperation) rollbackNode(ctx context.Context, state *controlPlaneNodeProgress) error {
	nodeName := state.snapshot.machineName
	var rollbackErrs []error
	// If resize didn't happen, there's nothing to restore.
	vmSizeRestored := !state.vmResized
	nodeReadyForSchedRestore := !state.vmStopped && !state.vmResized

	if state.vmResized {
		start := time.Now()
		err := func() error {
			// Rollback always deallocates (true) regardless of the forward path's
			// deallocateVM flag. Cross-family resizes require deallocation; during
			// rollback we take the most conservative Azure path to ensure restoring
			// the original SKU succeeds reliably.
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
		}()
		o.recordStep(nodeName, "restoreVMSize", time.Since(start), err)
		if err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.vmStopped = false
		}
	} else if state.vmStopped {
		nodeReadyForSchedRestore = false
		start := time.Now()
		startErr := retryAzureOperation(ctx, "start VM during rollback", func() error {
			return o.a.VMStartAndWait(ctx, nodeName)
		})
		o.recordStep(nodeName, "start", time.Since(start), startErr)
		if startErr != nil {
			rollbackErrs = append(rollbackErrs, startErr)
		} else {
			start = time.Now()
			waitErr := waitForNodeReady(ctx, o.log, o.k, nodeName)
			o.recordStep(nodeName, "waitReady", time.Since(start), waitErr)
			if waitErr != nil {
				rollbackErrs = append(rollbackErrs, waitErr)
			} else {
				start = time.Now()
				etcdErr := waitForEtcdHealthy(ctx, o.log, o.k)
				o.recordStep(nodeName, "waitEtcd", time.Since(start), etcdErr)
				if etcdErr != nil {
					rollbackErrs = append(rollbackErrs, etcdErr)
				} else {
					state.vmStopped = false
					nodeReadyForSchedRestore = true
				}
			}
		}
	}

	if state.machineUpdated && vmSizeRestored {
		start := time.Now()
		err := updateMachineVMSize(ctx, o.k, nodeName, state.snapshot.originalMachineSize)
		o.recordStep(nodeName, "restoreMachine", time.Since(start), err)
		if err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.machineUpdated = false
		}
	}

	if state.nodeLabelsUpdated && vmSizeRestored {
		start := time.Now()
		err := restoreNodeInstanceTypeLabels(ctx, o.k, nodeName, state.snapshot.originalNodeInstanceType, state.snapshot.originalNodeBetaInstanceType)
		o.recordStep(nodeName, "restoreNodeLabels", time.Since(start), err)
		if err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.nodeLabelsUpdated = false
		}
	}

	if state.schedulabilityNeedsRestore && nodeReadyForSchedRestore {
		start := time.Now()
		var err error
		if state.snapshot.originallySchedulable {
			err = uncordonNode(ctx, o.k, nodeName)
		} else {
			err = cordonNode(ctx, o.k, nodeName)
		}
		o.recordStep(nodeName, "restoreSchedulability", time.Since(start), err)
		if err != nil {
			rollbackErrs = append(rollbackErrs, err)
		} else {
			state.schedulabilityNeedsRestore = false
		}
	}

	return errors.Join(rollbackErrs...)
}

// rollbackAll processes nodes in reverse order and rechecks etcd between nodes.
// If etcd is unhealthy mid-rollback, stop immediately so SRE can take a targeted recovery path.
func (o *resizeControlPlaneOperation) rollbackAll(ctx context.Context) error {
	var errs []error
	for offset := range len(o.nodes) {
		i := len(o.nodes) - 1 - offset
		if i < len(o.nodes)-1 {
			if err := validateEtcdHealth(ctx, o.k); err != nil {
				return errors.Join(errors.Join(errs...), fmt.Errorf("etcd unhealthy before rollback of %s: %w", o.nodes[i].snapshot.machineName, err))
			}
		}
		if err := o.rollbackNode(ctx, o.nodes[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
