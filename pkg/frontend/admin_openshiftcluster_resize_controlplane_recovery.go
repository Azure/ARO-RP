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
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

const (
	resizeControlPlanePerNodeTimeout = 45 * time.Minute

	azureOperationMaxAttempts = 3
	azureOperationRetryDelay  = 5 * time.Second

	etcdHealthPollTimeout  = 10 * time.Minute
	etcdHealthPollInterval = 10 * time.Second
)

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
	machineName           string
	originalVMSize        string
	originalMachineSize   string
	originallySchedulable bool
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

// newResizeControlPlaneExecutionContext stays local because only the admin
// handler knows when resize work must outlive the incoming HTTP request.
// The resize operation (~45 min for 3 nodes) must complete even if the HTTP
// connection drops — the Geneva Actions client sets Timeout =
// InfiniteTimeSpan, but intermediary load balancers between the ACIS host and
// the RP VMSS can still close long-lived connections. ARM is not in this path;
// Geneva Actions calls the admin API directly via client certificate.
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

// recordStep stays local because rollback and admin replies rely on these
// operator-facing step names rather than pkg/util/steps' generic step labels.
func (o *resizeControlPlaneOperation) recordStep(node, step string, d time.Duration, err error) {
	if err != nil {
		o.steps = append(o.steps, fmt.Sprintf("%s:%s failed (%s): %v", node, step, d.Truncate(time.Millisecond), err))
	} else {
		o.steps = append(o.steps, fmt.Sprintf("%s:%s (%s)", node, step, d.Truncate(time.Millisecond)))
	}
}

func (o *resizeControlPlaneOperation) runStep(ctx context.Context, nodeName, stepName string, step steps.Step) error {
	start := time.Now()
	_, err := steps.Run(ctx, o.log, 0, []steps.Step{step}, nil, "")
	err = unwrapSyntheticStepRunnerError(err)
	o.recordStep(nodeName, stepName, time.Since(start), err)
	if err != nil {
		return fmt.Errorf("%s: %w", stepName, err)
	}
	return nil
}

func unwrapSyntheticStepRunnerError(err error) error {
	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) &&
		cloudErr.Target == "encountered error" &&
		(cloudErr.Code == api.CloudErrorCodeInvalidServicePrincipalCredentials ||
			cloudErr.Code == api.CloudErrorCodeInternalServerError) {
		return errors.New(cloudErr.Message)
	}

	return err
}

// captureNodeSnapshot stays local because rollback needs per-node state captured
// exactly at mutation time, not just generic step execution.
func (o *resizeControlPlaneOperation) captureNodeSnapshot(ctx context.Context, machineName string, machine machineValidationData) (controlPlaneNodeSnapshot, error) {
	vm, err := o.a.GetVirtualMachine(ctx, o.clusterResourceGroupName, machineName, mgmtcompute.InstanceView)
	if err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture Azure VM state for %s: %w", machineName, err)
	}
	if vm.VirtualMachineProperties == nil || vm.HardwareProfile == nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture Azure VM state for %s: HardwareProfile missing", machineName)
	}

	actualVMSize := string(vm.HardwareProfile.VMSize)

	rawNode, err := o.k.KubeGet(ctx, "Node", "", machineName)
	if err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to capture node state for %s: %w", machineName, err)
	}

	var node corev1.Node
	if err := json.Unmarshal(rawNode, &node); err != nil {
		return controlPlaneNodeSnapshot{}, fmt.Errorf("failed to parse node state for %s: %w", machineName, err)
	}

	snapshot := controlPlaneNodeSnapshot{
		machineName:           machineName,
		originalVMSize:        actualVMSize,
		originalMachineSize:   machine.size,
		originallySchedulable: !node.Spec.Unschedulable,
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

	type resizeNodeStep struct {
		name  string
		step  steps.Step
		after func()
	}

	stepEntries := make([]resizeNodeStep, 0, 9)
	if state.snapshot.originallySchedulable {
		stepEntries = append(stepEntries, resizeNodeStep{
			name: "cordon",
			step: steps.Action(func(ctx context.Context) error {
				return cordonNode(ctx, o.k, nodeName)
			}),
			after: func() {
				state.schedulabilityNeedsRestore = true
			},
		})
	}

	stepEntries = append(stepEntries,
		resizeNodeStep{
			name: "drain",
			step: steps.Action(func(ctx context.Context) error {
				return o.k.DrainNodeWithRetries(ctx, nodeName)
			}),
		},
		resizeNodeStep{
			name: "stop",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMStopAndWait(ctx, nodeName, o.deallocateVM)
			}),
			after: func() {
				state.vmStopped = true
			},
		},
		resizeNodeStep{
			name: "resize",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMResize(ctx, nodeName, o.desiredVMSize)
			}),
			after: func() {
				state.vmResized = true
			},
		},
		resizeNodeStep{
			name: "start",
			step: steps.Action(func(ctx context.Context) error {
				return o.a.VMStartAndWait(ctx, nodeName)
			}),
		},
		resizeNodeStep{
			name: "waitReady",
			step: steps.Action(func(ctx context.Context) error {
				return waitForNodeReady(ctx, o.log, o.k, nodeName)
			}),
			after: func() {
				state.vmStopped = false
			},
		},
		resizeNodeStep{
			name: "waitEtcd",
			step: steps.Action(func(ctx context.Context) error {
				return waitForEtcdHealthy(ctx, o.log, o.k)
			}),
		},
	)

	if state.snapshot.originallySchedulable {
		stepEntries = append(stepEntries, resizeNodeStep{
			name: "uncordon",
			step: steps.Action(func(ctx context.Context) error {
				return uncordonNode(ctx, o.k, nodeName)
			}),
			after: func() {
				state.schedulabilityNeedsRestore = false
			},
		})
	}

	stepEntries = append(stepEntries,
		resizeNodeStep{
			name: "updateMachine",
			step: steps.Action(func(ctx context.Context) error {
				return updateMachineVMSize(ctx, o.k, nodeName, o.desiredVMSize)
			}),
			after: func() {
				state.machineUpdated = true
			},
		},
		resizeNodeStep{
			name: "updateNodeLabels",
			step: steps.Action(func(ctx context.Context) error {
				return setNodeInstanceTypeLabels(ctx, o.k, nodeName, o.desiredVMSize)
			}),
			after: func() {
				state.nodeLabelsUpdated = true
			},
		},
	)

	for _, step := range stepEntries {
		if err := o.runStep(ctx, nodeName, step.name, step.step); err != nil {
			return err
		}
		if step.after != nil {
			step.after()
		}
	}

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

// Keep Azure retries local so resize and rollback keep the same semantics the
// recovery tests assert without pushing policy into pkg/util/steps.
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

// rollbackNode stays local because it replays stateful compensation based on
// partial progress flags that pkg/util/steps does not model.
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
		err := setNodeInstanceTypeLabels(ctx, o.k, nodeName, state.snapshot.originalMachineSize)
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

// rollbackAll stays local because it coordinates reverse-order compensation
// across nodes, including the etcd gate between rollback attempts.
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
