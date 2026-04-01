package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

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
	// getControlPlaneMachines filters by machine.openshift.io/cluster-api-machine-role=master,
	// so the returned map only contains control plane machines.
	machines, err := getControlPlaneMachines(ctx, k)
	if err != nil {
		return err
	}

	if len(machines) == 0 {
		return fmt.Errorf("no control plane machines found")
	}

	// Reverse lexicographic order: master-2 → master-1 → master-0.
	// This minimises etcd leader elections by resizing the highest-indexed
	// (conventionally least critical) node first, matching the C# behaviour.
	sortedNames := slices.SortedFunc(maps.Keys(machines), func(a, b string) int {
		return cmp.Compare(b, a)
	})

	for _, name := range sortedNames {
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
func resizeControlPlaneNode(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, a adminactions.AzureActions, machineName, desiredVMSize string, deallocateVM bool) error {
	log.Infof("Cordoning node %s", machineName)
	if err := cordonNode(ctx, k, machineName); err != nil {
		return fmt.Errorf("cordoning node: %w", err)
	}

	log.Infof("Draining node %s", machineName)
	if err := k.DrainNodeWithRetries(ctx, machineName); err != nil {
		return fmt.Errorf("draining node: %w", err)
	}

	log.Infof("Stopping VM %s (deallocate=%v)", machineName, deallocateVM)
	if err := a.VMStopAndWait(ctx, machineName, deallocateVM); err != nil {
		return fmt.Errorf("stopping VM: %w", err)
	}

	log.Infof("Resizing VM %s to %s", machineName, desiredVMSize)
	if err := a.VMResize(ctx, machineName, desiredVMSize); err != nil {
		return fmt.Errorf("resizing VM: %w", err)
	}

	log.Infof("Starting VM %s", machineName)
	if err := a.VMStartAndWait(ctx, machineName); err != nil {
		return fmt.Errorf("starting VM: %w", err)
	}

	log.Infof("Waiting for node %s to become Ready", machineName)
	if err := waitForNodeReady(ctx, log, k, machineName); err != nil {
		return fmt.Errorf("waiting for node ready: %w", err)
	}

	log.Infof("Uncordoning node %s", machineName)
	if err := uncordonNode(ctx, k, machineName); err != nil {
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

func cordonNode(ctx context.Context, k adminactions.KubeActions, nodeName string) error {
	return k.CordonNode(ctx, nodeName, true)
}

func uncordonNode(ctx context.Context, k adminactions.KubeActions, nodeName string) error {
	return k.CordonNode(ctx, nodeName, false)
}

// getControlPlaneMachines is a thin wrapper around getClusterMachines that
// makes the intent explicit at the call site. getClusterMachines already
// filters by the machine.openshift.io/cluster-api-machine-role=master label.
func getControlPlaneMachines(ctx context.Context, k adminactions.KubeActions) (map[string]machineValidationData, error) {
	return getClusterMachines(ctx, k)
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
	ctx, cancel := context.WithTimeout(ctx, nodeReadyPollTimeout)
	defer cancel()

	return wait.PollImmediateUntilWithContext(ctx, nodeReadyPollInterval, func(ctx context.Context) (bool, error) {
		ready, err := isNodeReady(ctx, k, nodeName)
		if err != nil {
			log.Infof("Error checking node %s readiness: %v", nodeName, err)
			return false, nil
		}
		if !ready {
			log.Infof("Waiting for node %s to become Ready...", nodeName)
		}
		return ready, nil
	})
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

	var machine machinev1beta1.Machine
	if err := json.Unmarshal(rawMachine, &machine); err != nil {
		return err
	}

	providerSpec := &machinev1beta1.AzureMachineProviderSpec{}
	if err := json.Unmarshal(machine.Spec.ProviderSpec.Value.Raw, providerSpec); err != nil {
		return fmt.Errorf("parsing providerSpec: %w", err)
	}

	providerSpec.VMSize = vmSize

	rawProviderSpec, err := json.Marshal(providerSpec)
	if err != nil {
		return fmt.Errorf("marshalling providerSpec: %w", err)
	}

	machine.Spec.ProviderSpec.Value.Raw = rawProviderSpec

	if machine.Labels == nil {
		machine.Labels = make(map[string]string)
	}
	machine.Labels[machineLabelInstanceType] = vmSize

	rawBytes, err := json.Marshal(&machine)
	if err != nil {
		return fmt.Errorf("marshalling machine: %w", err)
	}
	var obj unstructured.Unstructured
	if err := json.Unmarshal(rawBytes, &obj.Object); err != nil {
		return fmt.Errorf("converting to unstructured: %w", err)
	}

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
