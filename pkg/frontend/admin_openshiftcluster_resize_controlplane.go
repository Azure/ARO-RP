package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

const (
	nodeReadyPollTimeout       = 30 * time.Minute
	nodeReadyPollInterval      = 5 * time.Second
	kubeObjectUpdateMaxRetries = 3
	kubeObjectUpdateRetryDelay = time.Second
)

func (f *frontend) postAdminResizeControlPlane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminResizeControlPlane(log, ctx, r)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminResizeControlPlane(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	resType := chi.URLParam(r, "resourceType")
	resName := chi.URLParam(r, "resourceName")
	resGroupName := chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	vmSize := r.URL.Query().Get("vmSize")
	deallocateVM := true
	if v := r.URL.Query().Get("deallocateVM"); v != "" {
		deallocateVM = strings.EqualFold(v, "true")
	}

	if err := validateAdminMasterVMSize(vmSize); err != nil {
		return err
	}

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return err
	}

	// Run all pre-flight validations (API server health, etcd health, SP, VM SKU, quota).
	_, err = f._getPreResizeControlPlaneVMsValidation(ctx, resType, resName, resGroupName, resourceID, vmSize, log)
	if err != nil {
		return err
	}

	return resizeControlPlane(ctx, log, k, a, vmSize, deallocateVM)
}

// resizeControlPlane orchestrates the full control plane resize operation,
// processing each master node sequentially in reverse name order.
func resizeControlPlane(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, a adminactions.AzureActions, desiredVMSize string, deallocateVM bool) error {
	if err := checkCPMSNotActive(ctx, k); err != nil {
		return err
	}

	machines, err := getClusterMachines(ctx, k)
	if err != nil {
		return err
	}

	if len(machines) == 0 {
		return fmt.Errorf("no control plane machines found")
	}

	machineNames := make([]string, 0, len(machines))
	for name := range machines {
		machineNames = append(machineNames, name)
	}
	// Process in reverse lexicographic order so the highest-numbered master
	// (conventionally the least critical, e.g. master-2) is resized first.
	sort.Sort(sort.Reverse(sort.StringSlice(machineNames)))

	for _, name := range machineNames {
		machine := machines[name]
		if machine.size == desiredVMSize {
			log.Infof("%s is already running %s, skipping", name, desiredVMSize)
			continue
		}

		log.Infof("Resizing control plane node %s from %s to %s", name, machine.size, desiredVMSize)
		if err := resizeControlPlaneNode(ctx, log, k, a, name, desiredVMSize, deallocateVM); err != nil {
			return fmt.Errorf("failed to resize node %s: %w", name, err)
		}
		log.Infof("Successfully resized node %s to %s", name, desiredVMSize)
	}

	return nil
}

// resizeControlPlaneNode performs the full resize sequence for a single
// control plane node: cordon → drain → stop → resize → start → wait
// ready → uncordon → update Machine metadata → update Node labels.
//
// If a failure occurs before the VM SKU has been changed (drain, stop,
// or resize), best-effort recovery is attempted to restore the node to a
// schedulable state. Failures after the SKU change (start, wait-ready)
// are reported without automatic recovery — SRE should intervene per SOP.
func resizeControlPlaneNode(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, a adminactions.AzureActions, machineName, desiredVMSize string, deallocateVM bool) error {
	log.Infof("Cordoning node %s", machineName)
	if err := k.CordonNode(ctx, machineName, true); err != nil {
		return fmt.Errorf("cordoning node: %w", err)
	}

	log.Infof("Draining node %s", machineName)
	if err := k.DrainNodeWithRetries(ctx, machineName); err != nil {
		recoveryErr := bestEffortUncordon(ctx, log, k, machineName)
		return resizeRecoveryError("draining node", err, recoveryErr)
	}

	log.Infof("Stopping VM %s (deallocate=%v)", machineName, deallocateVM)
	if err := a.VMStopAndWait(ctx, machineName, deallocateVM); err != nil {
		recoveryErr := bestEffortRecoverVM(ctx, log, k, a, machineName)
		return resizeRecoveryError("stopping VM", err, recoveryErr)
	}

	log.Infof("Resizing VM %s to %s", machineName, desiredVMSize)
	if err := a.VMResize(ctx, machineName, desiredVMSize); err != nil {
		recoveryErr := bestEffortRecoverVM(ctx, log, k, a, machineName)
		return resizeRecoveryError("resizing VM", err, recoveryErr)
	}

	// Past this point the VM SKU has been changed. No automatic size
	// rollback is attempted — the new size is the intended outcome.
	log.Infof("Starting VM %s", machineName)
	if err := a.VMStartAndWait(ctx, machineName); err != nil {
		return fmt.Errorf("starting VM: %w", err)
	}

	log.Infof("Waiting for node %s to become Ready", machineName)
	if err := waitForNodeReady(ctx, log, k, machineName); err != nil {
		return fmt.Errorf("waiting for node ready: %w", err)
	}

	log.Infof("Uncordoning node %s", machineName)
	if err := k.CordonNode(ctx, machineName, false); err != nil {
		return fmt.Errorf("uncordoning node: %w", err)
	}

	log.Infof("Updating Machine object for %s", machineName)
	if err := updateMachineVMSize(ctx, k, machineName, desiredVMSize); err != nil {
		return fmt.Errorf("updating Machine object: %w", err)
	}

	log.Infof("Updating Node labels for %s", machineName)
	if err := updateNodeInstanceTypeLabels(ctx, k, machineName, desiredVMSize); err != nil {
		return fmt.Errorf("updating Node labels: %w", err)
	}

	return nil
}

// bestEffortUncordon attempts to uncordon a node that is still running.
// Used when a failure occurs before the VM was stopped.
func bestEffortUncordon(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, machineName string) error {
	log.Infof("Recovery: attempting to uncordon node %s", machineName)
	if err := k.CordonNode(ctx, machineName, false); err != nil {
		log.Errorf("Recovery: failed to uncordon node %s: %v", machineName, err)
		return fmt.Errorf("recovery uncordon failed: %w", err)
	}
	log.Infof("Recovery: successfully uncordoned node %s", machineName)
	return nil
}

// bestEffortRecoverVM attempts to start a stopped VM, wait for the node to
// become Ready, and uncordon it. If the VM cannot be started, recovery stops.
// If the node does not become Ready within the timeout, uncordon is NOT
// attempted — per SOP, SRE should verify node health before re-enabling
// scheduling on a node whose health has not been confirmed.
func bestEffortRecoverVM(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, a adminactions.AzureActions, machineName string) error {
	log.Infof("Recovery: attempting to start VM %s", machineName)
	if err := a.VMStartAndWait(ctx, machineName); err != nil {
		log.Errorf("Recovery: failed to start VM %s: %v", machineName, err)
		return fmt.Errorf("recovery VM start failed: %w", err)
	}

	log.Infof("Recovery: VM %s started, waiting for node to become Ready", machineName)
	if err := waitForNodeReady(ctx, log, k, machineName); err != nil {
		log.Errorf("Recovery: node %s did not become Ready: %v. Node left cordoned per SOP — SRE should verify node health.", machineName, err)
		return fmt.Errorf("recovery wait-for-ready failed (node left cordoned per SOP): %w", err)
	}

	log.Infof("Recovery: node %s is Ready, uncordoning", machineName)
	if err := k.CordonNode(ctx, machineName, false); err != nil {
		log.Errorf("Recovery: failed to uncordon node %s: %v", machineName, err)
		return fmt.Errorf("recovery uncordon failed: %w", err)
	}

	log.Infof("Recovery: node %s fully recovered", machineName)
	return nil
}

// resizeRecoveryError combines the original resize failure with the
// recovery outcome so SREs can see both in a single error message.
func resizeRecoveryError(operation string, resizeErr, recoveryErr error) error {
	if recoveryErr != nil {
		return fmt.Errorf("%s: %w; recovery also failed: %v", operation, resizeErr, recoveryErr)
	}
	return fmt.Errorf("%s: %w; node was recovered successfully", operation, resizeErr)
}

// checkCPMSNotActive verifies that the ControlPlaneMachineSet is not Active.
// If it is active, direct VM manipulation would conflict with the CPMS operator.
// Only NotFound / CRD-not-installed errors are treated as "CPMS absent";
// all other errors fail the operation closed so we don't bypass the safety check.
func checkCPMSNotActive(ctx context.Context, k adminactions.KubeActions) error {
	rawCPMS, err := k.KubeGet(ctx, "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster")
	if err != nil {
		if kerrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("failed to check ControlPlaneMachineSet state: %v", err))
	}

	var obj unstructured.Unstructured
	if err := json.Unmarshal(rawCPMS, &obj); err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("failed to parse ControlPlaneMachineSet object: %v", err))
	}

	state, found, err := unstructured.NestedString(obj.Object, "spec", "state")
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("failed to read ControlPlaneMachineSet state: %v", err))
	}
	if found && strings.EqualFold(state, "Active") {
		return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "",
			"ControlPlaneMachineSet is currently Active. Deactivate CPMS before running this operation.")
	}

	return nil
}

func waitForNodeReady(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, nodeName string) error {
	deadline := time.Now().Add(nodeReadyPollTimeout)

	for {
		ready, err := isNodeReady(ctx, k, nodeName)
		if err != nil {
			log.Infof("Error checking node %s readiness: %v", nodeName, err)
		} else if ready {
			return nil
		} else {
			log.Infof("Waiting for node %s to become Ready...", nodeName)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("node %s did not become Ready within %v", nodeName, nodeReadyPollTimeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(nodeReadyPollInterval):
		}
	}
}

func isNodeReady(ctx context.Context, k adminactions.KubeActions, nodeName string) (bool, error) {
	rawNode, err := k.KubeGet(ctx, "Node", "", nodeName)
	if err != nil {
		return false, err
	}

	var node unstructured.Unstructured
	if err := json.Unmarshal(rawNode, &node); err != nil {
		return false, err
	}

	conditions, found, err := unstructured.NestedSlice(node.Object, "status", "conditions")
	if err != nil || !found {
		return false, nil
	}

	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := cond["type"].(string)
		condStatus, _ := cond["status"].(string)
		if condType == "Ready" {
			return condStatus == "True", nil
		}
	}

	return false, nil
}

func updateMachineVMSize(ctx context.Context, k adminactions.KubeActions, machineName, vmSize string) error {
	var lastErr error
	for attempt := 0; attempt <= kubeObjectUpdateMaxRetries; attempt++ {
		lastErr = doUpdateMachineVMSize(ctx, k, machineName, vmSize)
		if lastErr == nil {
			return nil
		}
		if attempt < kubeObjectUpdateMaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(kubeObjectUpdateRetryDelay):
			}
		}
	}
	return fmt.Errorf("could not update Machine object after %d retries: %w", kubeObjectUpdateMaxRetries, lastErr)
}

func doUpdateMachineVMSize(ctx context.Context, k adminactions.KubeActions, machineName, vmSize string) error {
	rawMachine, err := k.KubeGet(ctx, "Machine.machine.openshift.io", machineNamespace, machineName)
	if err != nil {
		return err
	}

	var obj unstructured.Unstructured
	if err := json.Unmarshal(rawMachine, &obj); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(obj.Object, vmSize, "spec", "providerSpec", "value", "vmSize"); err != nil {
		return fmt.Errorf("setting vmSize in providerSpec: %w", err)
	}

	// Sync the providerSpec embedded metadata.creationTimestamp with the
	// machine's own creationTimestamp to satisfy API validation.
	ts, found, err := unstructured.NestedString(obj.Object, "metadata", "creationTimestamp")
	if err != nil {
		return fmt.Errorf("reading machine creationTimestamp: %w", err)
	}
	if found {
		if err := unstructured.SetNestedField(obj.Object, ts, "spec", "providerSpec", "value", "metadata", "creationTimestamp"); err != nil {
			return fmt.Errorf("setting metadata.creationTimestamp in providerSpec: %w", err)
		}
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[machineLabelInstanceType] = vmSize
	obj.SetLabels(labels)

	delete(obj.Object, "status")

	return k.KubeCreateOrUpdate(ctx, &obj)
}

func updateNodeInstanceTypeLabels(ctx context.Context, k adminactions.KubeActions, nodeName, vmSize string) error {
	var lastErr error
	for attempt := 0; attempt <= kubeObjectUpdateMaxRetries; attempt++ {
		lastErr = doUpdateNodeInstanceTypeLabels(ctx, k, nodeName, vmSize)
		if lastErr == nil {
			return nil
		}
		if attempt < kubeObjectUpdateMaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(kubeObjectUpdateRetryDelay):
			}
		}
	}
	return fmt.Errorf("could not update Node object after %d retries: %w", kubeObjectUpdateMaxRetries, lastErr)
}

func doUpdateNodeInstanceTypeLabels(ctx context.Context, k adminactions.KubeActions, nodeName, vmSize string) error {
	rawNode, err := k.KubeGet(ctx, "Node", "", nodeName)
	if err != nil {
		return err
	}

	var obj unstructured.Unstructured
	if err := json.Unmarshal(rawNode, &obj); err != nil {
		return err
	}

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[nodeLabelInstanceType] = vmSize
	labels[nodeLabelBetaInstanceType] = vmSize
	obj.SetLabels(labels)

	delete(obj.Object, "status")

	return k.KubeCreateOrUpdate(ctx, &obj)
}
