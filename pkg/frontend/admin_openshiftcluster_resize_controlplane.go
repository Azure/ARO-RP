package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	adminapi "github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

const (
	nodeReadyPollTimeout        = 30 * time.Minute
	nodeReadyPollInterval       = 5 * time.Second
	kubeObjectUpdateMaxAttempts = 3
	kubeObjectUpdateRetryDelay  = time.Second
)

func (f *frontend) postAdminResizeControlPlane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._postAdminResizeControlPlane(log, ctx, r)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _postAdminResizeControlPlane(log *logrus.Entry, ctx context.Context, r *http.Request) ([]byte, error) {
	operationStart := time.Now()
	resType := chi.URLParam(r, "resourceType")
	resName := chi.URLParam(r, "resourceName")
	resGroupName := chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	vmSize := r.URL.Query().Get("vmSize")
	deallocateVM := true
	if v := r.URL.Query().Get("deallocateVM"); v != "" {
		switch {
		case strings.EqualFold(v, "true"):
			deallocateVM = true
		case strings.EqualFold(v, "false"):
			deallocateVM = false
		default:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "deallocateVM",
				fmt.Sprintf("The provided deallocateVM value '%s' is invalid. Allowed values are 'true' or 'false'.", v))
		}
	}

	if err := validateAdminMasterVMSize(vmSize); err != nil {
		return nil, err
	}

	report := newResizeControlPlaneResponse(resourceID, vmSize, deallocateVM)

	var (
		doc             *api.OpenShiftClusterDocument
		subscriptionDoc *api.SubscriptionDocument
		k               adminactions.KubeActions
		a               adminactions.AzureActions
	)

	err := runResizePhase(report, "request-setup", func(phase *adminapi.ResizeControlPlanePhase) error {
		dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
		if err != nil {
			return err
		}

		doc, err = dbOpenShiftClusters.Get(ctx, resourceID)
		switch {
		case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
			return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
				fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
		case err != nil:
			return err
		}

		report.ResourceID = doc.OpenShiftCluster.ID

		subscriptionDoc, err = f.getSubscriptionDocument(ctx, doc.Key)
		if err != nil {
			return err
		}

		k, err = f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
		if err != nil {
			return err
		}

		a, err = f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
		if err != nil {
			return err
		}

		phase.Message = fmt.Sprintf("Loaded cluster documents and initialized Kubernetes and Azure action clients for cluster location %s.",
			doc.OpenShiftCluster.Location)
		return nil
	})
	if err != nil {
		report.DurationMS = time.Since(operationStart).Milliseconds()
		return nil, buildResizeControlPlaneCloudError(report, wrapResizeOperationError(
			"request-setup",
			"",
			"",
			"Check RP access to the cluster document, subscription document, and admin action client initialization.",
			err))
	}

	// Run all pre-flight validations (API server health, etcd health, SP, VM SKU, quota).
	preflightStart := time.Now()
	preflightResult, err := f.runPreResizeControlPlaneVMsValidation(ctx, doc, subscriptionDoc, k, a, vmSize)
	preflightPhase := adminapi.ResizeControlPlanePhase{
		Name:       "pre-flight-validation",
		Status:     adminapi.ResizeControlPlaneOperationStatusSucceeded,
		DurationMS: time.Since(preflightStart).Milliseconds(),
		Message: fmt.Sprintf("Validated %d pre-flight check(s) before starting control plane changes.",
			len(preflightResult.Checks)),
		Checks: preflightResult.Checks,
	}
	if err != nil {
		preflightPhase.Status = adminapi.ResizeControlPlaneOperationStatusFailed
		preflightPhase.Message = fmt.Sprintf("Pre-flight validation failed after %s.",
			formatResizeDuration(time.Duration(preflightPhase.DurationMS)*time.Millisecond))
		report.Phases = append(report.Phases, preflightPhase)
		report.DurationMS = time.Since(operationStart).Milliseconds()
		return nil, buildResizeControlPlaneCloudError(report, wrapResizeOperationError(
			"pre-flight-validation",
			"",
			"",
			"Resolve the validation failures listed in details before retrying the resize.",
			err))
	}
	report.Phases = append(report.Phases, preflightPhase)

	err = resizeControlPlaneWithReport(ctx, log, k, a, vmSize, deallocateVM, report)
	if err != nil {
		report.DurationMS = time.Since(operationStart).Milliseconds()
		return nil, buildResizeControlPlaneCloudError(report, err)
	}

	report.DurationMS = time.Since(operationStart).Milliseconds()
	report.Message = fmt.Sprintf(
		"Control plane resize completed successfully in %s. Processed %d node(s): resized %d and skipped %d. Each resized node was cordoned, drained, stopped, resized, started, verified Ready, uncordoned, and had Machine/Node metadata updated.",
		formatResizeDuration(time.Since(operationStart)),
		report.Summary.TotalNodes,
		report.Summary.NodesResized,
		report.Summary.NodesSkipped,
	)

	b, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func resizeControlPlaneWithReport(
	ctx context.Context,
	log *logrus.Entry,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
	desiredVMSize string,
	deallocateVM bool,
	report *adminapi.ResizeControlPlaneResponse,
) error {
	var machines map[string]machineValidationData

	err := runResizePhase(report, "discover-control-plane-machines", func(phase *adminapi.ResizeControlPlanePhase) error {
		var err error
		// getControlPlaneMachines filters by machine.openshift.io/cluster-api-machine-role=master,
		// so the returned map only contains control plane machines.
		machines, err = getControlPlaneMachines(ctx, k)
		if err != nil {
			return wrapResizeOperationError(
				"discover-control-plane-machines",
				"",
				"",
				"Check Machine API availability and the control plane Machine resources in openshift-machine-api.",
				err,
			)
		}

		if len(machines) == 0 {
			return wrapResizeOperationError(
				"discover-control-plane-machines",
				"",
				"",
				"Check that control plane Machine resources exist in openshift-machine-api before retrying.",
				api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "",
					"No control plane machines found. Resize cannot proceed."),
			)
		}

		phase.Message = fmt.Sprintf("Discovered %d control plane machine(s).", len(machines))
		return nil
	})
	if err != nil {
		return err
	}

	// Reverse lexicographic order: master-2 → master-1 → master-0.
	// This minimises etcd leader elections by resizing the highest-indexed
	// (conventionally least critical) node first, matching the C# behaviour.
	sortedNames := slices.SortedFunc(maps.Keys(machines), func(a, b string) int {
		return cmp.Compare(b, a)
	})
	if report != nil {
		report.Summary.TotalNodes = len(sortedNames)
		report.Summary.ExecutionOrder = append(report.Summary.ExecutionOrder[:0], sortedNames...)
	}

	err = runResizePhase(report, "verify-control-plane-health", func(phase *adminapi.ResizeControlPlanePhase) error {
		// Guard the whole operation before touching any VM. Even when a machine
		// already matches the target SKU, we must not continue to the next resize
		// while another control plane node is NotReady or still cordoned.
		if err := ensureControlPlaneNodesReadyAndSchedulable(ctx, k, sortedNames); err != nil {
			return wrapResizeOperationError(
				"verify-control-plane-health",
				"",
				"",
				"Ensure every control plane node is Ready and schedulable before retrying the resize.",
				err,
			)
		}

		phase.Message = fmt.Sprintf("Verified that all %d control plane node(s) were Ready and schedulable before resize started.",
			len(sortedNames))
		return nil
	})
	if err != nil {
		return err
	}

	phaseStart := time.Now()
	resizePhase := adminapi.ResizeControlPlanePhase{
		Name:   "resize-control-plane-nodes",
		Status: adminapi.ResizeControlPlaneOperationStatusSucceeded,
	}
	nodesResized := 0
	nodesSkipped := 0

	for _, name := range sortedNames {
		machine := machines[name]
		nodeResult := adminapi.ResizeControlPlaneNodeOperation{
			Name:         name,
			SourceVMSize: machine.size,
			TargetVMSize: desiredVMSize,
		}
		nodeStart := time.Now()

		if machine.size == desiredVMSize {
			log.Infof("%s is already running %s, skipping", name, desiredVMSize)
			nodeResult.Status = adminapi.ResizeControlPlaneOperationStatusSkipped
			nodeResult.DurationMS = time.Since(nodeStart).Milliseconds()
			nodeResult.Message = "Node already running target VM size; no resize was required."
			nodesSkipped++
			if report != nil {
				report.Nodes = append(report.Nodes, nodeResult)
				report.Summary.NodesSkipped = nodesSkipped
			}
			continue
		}

		log.Infof("Resizing control plane node %s from %s to %s", name, machine.size, desiredVMSize)
		err := resizeControlPlaneNodeWithReport(ctx, log, k, a, name, desiredVMSize, deallocateVM, &nodeResult)
		nodeResult.DurationMS = time.Since(nodeStart).Milliseconds()
		if err != nil {
			nodeResult.Status = adminapi.ResizeControlPlaneOperationStatusFailed
			nodeResult.Message = fmt.Sprintf("Resize failed for node %s. %s", name, resizeDiagnosticMessage(err))
			if report != nil {
				report.Nodes = append(report.Nodes, nodeResult)
			}
			resizePhase.Status = adminapi.ResizeControlPlaneOperationStatusFailed
			resizePhase.DurationMS = time.Since(phaseStart).Milliseconds()
			resizePhase.Message = fmt.Sprintf(
				"Resize failed while processing node %s after resizing %d node(s) and skipping %d node(s).",
				name,
				nodesResized,
				nodesSkipped,
			)
			if report != nil {
				report.Phases = append(report.Phases, resizePhase)
			}
			return fmt.Errorf("failed to resize node %s: %w", name, err)
		}

		log.Infof("Successfully resized node %s to %s", name, desiredVMSize)
		nodeResult.Status = adminapi.ResizeControlPlaneOperationStatusSucceeded
		nodeResult.Message = fmt.Sprintf(
			"Node resized from %s to %s and post-resize metadata updates completed.",
			machine.size,
			desiredVMSize,
		)
		nodesResized++
		if report != nil {
			report.Nodes = append(report.Nodes, nodeResult)
			report.Summary.NodesResized = nodesResized
		}
	}

	resizePhase.DurationMS = time.Since(phaseStart).Milliseconds()
	resizePhase.Message = fmt.Sprintf(
		"Processed %d control plane node(s) in reverse name order. Resized %d node(s) and skipped %d node(s).",
		len(sortedNames),
		nodesResized,
		nodesSkipped,
	)
	if report != nil {
		report.Phases = append(report.Phases, resizePhase)
	}

	return nil
}

// resizeControlPlaneNode performs the full resize sequence for a single
// control plane node: cordon → drain → stop → resize → start → wait
// ready → uncordon → update Machine metadata → update Node labels.
func resizeControlPlaneNodeWithReport(
	ctx context.Context,
	log *logrus.Entry,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
	machineName, desiredVMSize string,
	deallocateVM bool,
	nodeResult *adminapi.ResizeControlPlaneNodeOperation,
) error {
	recordStep := func(name, failurePrefix, successMessage, failureHint string, fn func() error) error {
		stepStart := time.Now()
		err := fn()
		step := adminapi.ResizeControlPlaneStep{
			Name:       name,
			DurationMS: time.Since(stepStart).Milliseconds(),
		}
		if err != nil {
			step.Status = adminapi.ResizeControlPlaneOperationStatusFailed
			step.Message = resizeDiagnosticMessage(err)
			nodeResult.Steps = append(nodeResult.Steps, step)
			return wrapResizeOperationError("resize-control-plane-nodes", machineName, name, failureHint,
				fmt.Errorf("%s: %w", failurePrefix, err))
		}

		step.Status = adminapi.ResizeControlPlaneOperationStatusSucceeded
		step.Message = successMessage
		nodeResult.Steps = append(nodeResult.Steps, step)
		return nil
	}

	log.Infof("Cordoning node %s", machineName)
	if err := recordStep("cordon", "cordoning node", "Cordoned node successfully.",
		"Check Kubernetes API connectivity and node object permissions before retrying.", func() error {
			return cordonNode(ctx, k, machineName)
		}); err != nil {
		return err
	}

	log.Infof("Draining node %s", machineName)
	if err := recordStep("drain", "draining node", "Drained node successfully.",
		"Check for PodDisruptionBudgets or workloads that prevented eviction on the node.", func() error {
			return k.DrainNodeWithRetries(ctx, machineName)
		}); err != nil {
		return err
	}

	log.Infof("Stopping VM %s (deallocate=%v)", machineName, deallocateVM)
	if err := recordStep("stop-vm", "stopping VM", fmt.Sprintf("Stopped VM successfully (deallocate=%v).", deallocateVM),
		"Check Azure Compute activity for VM stop failures or deallocation problems.", func() error {
			return a.VMStopAndWait(ctx, machineName, deallocateVM)
		}); err != nil {
		return err
	}

	log.Infof("Resizing VM %s to %s", machineName, desiredVMSize)
	if err := recordStep("resize-vm", "resizing VM", fmt.Sprintf("Resized VM to %s successfully.", desiredVMSize),
		"Check Azure Compute activity for resize or allocation failures on the VM.", func() error {
			return a.VMResize(ctx, machineName, desiredVMSize)
		}); err != nil {
		return err
	}

	log.Infof("Starting VM %s", machineName)
	if err := recordStep("start-vm", "starting VM", "Started VM successfully.",
		"Check Azure Compute activity for VM start failures.", func() error {
			return a.VMStartAndWait(ctx, machineName)
		}); err != nil {
		return err
	}

	log.Infof("Waiting for node %s to become Ready", machineName)
	if err := recordStep("wait-for-node-ready", "waiting for node ready", "Node reported Ready after restart.",
		"Check kubelet startup, node conditions, and control plane component health on the node.", func() error {
			return waitForNodeReady(ctx, log, k, machineName)
		}); err != nil {
		return err
	}

	log.Infof("Uncordoning node %s", machineName)
	if err := recordStep("uncordon", "uncordoning node", "Uncordoned node successfully.",
		"Check Kubernetes API connectivity and node schedulability before retrying.", func() error {
			return uncordonNode(ctx, k, machineName)
		}); err != nil {
		return err
	}

	log.Infof("Updating Machine object for %s", machineName)
	if err := recordStep("update-machine-object", "updating Machine object", "Updated Machine object with the new VM size.",
		"Check Machine API reconciliation and conflicting updates to the Machine resource.", func() error {
			return updateMachineVMSize(ctx, k, machineName, desiredVMSize)
		}); err != nil {
		return err
	}

	log.Infof("Updating Node labels for %s", machineName)
	if err := recordStep("update-node-labels", "updating Node labels", "Updated Node instance-type labels to the new VM size.",
		"Check Kubernetes API write access and conflicting updates to the Node resource.", func() error {
			return updateNodeInstanceTypeLabels(ctx, k, machineName, desiredVMSize)
		}); err != nil {
		return err
	}

	return nil
}

// resizeControlPlane orchestrates the full control plane resize operation,
// processing each master node sequentially in reverse name order.
func resizeControlPlane(ctx context.Context, log *logrus.Entry, k adminactions.KubeActions, a adminactions.AzureActions, desiredVMSize string, deallocateVM bool) error {
	return resizeControlPlaneWithReport(ctx, log, k, a, desiredVMSize, deallocateVM, nil)
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

	var cpms machinev1.ControlPlaneMachineSet
	if err := json.Unmarshal(rawCPMS, &cpms); err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("failed to parse ControlPlaneMachineSet object: %v", err))
	}

	if cpms.Spec.State == machinev1.ControlPlaneMachineSetStateActive {
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

func ensureControlPlaneNodesReadyAndSchedulable(ctx context.Context, k adminactions.KubeActions, nodeNames []string) error {
	for _, nodeName := range nodeNames {
		ready, schedulable, err := getNodeReadinessAndSchedulability(ctx, k, nodeName)
		if err != nil {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
				fmt.Sprintf("failed to evaluate control plane node %s health before resize: %v", nodeName, err))
		}
		if !ready {
			return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "",
				fmt.Sprintf("Control plane node %s is not Ready. Resolve node health before resizing another master.", nodeName))
		}
		if !schedulable {
			return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeRequestNotAllowed, "",
				fmt.Sprintf("Control plane node %s is unschedulable. Uncordon and verify the node before resizing another master.", nodeName))
		}
	}
	return nil
}

func isNodeReady(ctx context.Context, k adminactions.KubeActions, nodeName string) (bool, error) {
	ready, _, err := getNodeReadinessAndSchedulability(ctx, k, nodeName)
	return ready, err
}

func getNodeReadinessAndSchedulability(ctx context.Context, k adminactions.KubeActions, nodeName string) (bool, bool, error) {
	rawNode, err := k.KubeGet(ctx, "Node", "", nodeName)
	if err != nil {
		return false, false, err
	}

	var node corev1.Node
	if err := json.Unmarshal(rawNode, &node); err != nil {
		return false, false, err
	}

	schedulable := !node.Spec.Unschedulable
	ready := false

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			ready = condition.Status == corev1.ConditionTrue
			break
		}
	}

	return ready, schedulable, nil
}

func updateMachineVMSize(ctx context.Context, k adminactions.KubeActions, machineName, vmSize string) error {
	return retryKubeObjectUpdate(ctx, "Machine", func() error {
		return doUpdateMachineVMSize(ctx, k, machineName, vmSize)
	})
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
	providerSpec.SetCreationTimestamp(machine.GetCreationTimestamp())

	rawProviderSpec, err := json.Marshal(providerSpec)
	if err != nil {
		return fmt.Errorf("marshalling providerSpec: %w", err)
	}

	machine.Spec.ProviderSpec.Value.Raw = rawProviderSpec

	if machine.Labels == nil {
		machine.Labels = make(map[string]string)
	}
	machine.Labels[machineLabelInstanceType] = vmSize

	// KubeCreateOrUpdate expects unstructured objects, so convert the typed Machine before nested field updates.
	objMap, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(&machine)
	if err != nil {
		return fmt.Errorf("converting machine to unstructured: %w", err)
	}
	obj := unstructured.Unstructured{Object: objMap}

	delete(obj.Object, "status")

	return k.KubeCreateOrUpdate(ctx, &obj)
}

func updateNodeInstanceTypeLabels(ctx context.Context, k adminactions.KubeActions, nodeName, vmSize string) error {
	return retryKubeObjectUpdate(ctx, "Node", func() error {
		return doUpdateNodeInstanceTypeLabels(ctx, k, nodeName, vmSize)
	})
}

func retryKubeObjectUpdate(ctx context.Context, objectType string, updateFn func() error) error {
	var lastErr error
	for attempt := range kubeObjectUpdateMaxAttempts {
		lastErr = updateFn()
		if lastErr == nil {
			return nil
		}

		if attempt == kubeObjectUpdateMaxAttempts-1 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(kubeObjectUpdateRetryDelay):
		}
	}

	return fmt.Errorf("could not update %s object after %d attempts: %w", objectType, kubeObjectUpdateMaxAttempts, lastErr)
}

func doUpdateNodeInstanceTypeLabels(ctx context.Context, k adminactions.KubeActions, nodeName, vmSize string) error {
	rawNode, err := k.KubeGet(ctx, "Node", "", nodeName)
	if err != nil {
		return err
	}

	var node corev1.Node
	if err := json.Unmarshal(rawNode, &node); err != nil {
		return err
	}

	labels := node.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[nodeLabelInstanceType] = vmSize
	labels[nodeLabelBetaInstanceType] = vmSize
	node.SetLabels(labels)

	objMap, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(&node)
	if err != nil {
		return fmt.Errorf("converting node to unstructured: %w", err)
	}
	obj := unstructured.Unstructured{Object: objMap}

	delete(obj.Object, "status")

	return k.KubeCreateOrUpdate(ctx, &obj)
}

type resizeOperationError struct {
	phase string
	node  string
	step  string
	hint  string
	err   error
}

func (e *resizeOperationError) Error() string {
	return e.err.Error()
}

func (e *resizeOperationError) Unwrap() error {
	return e.err
}

func wrapResizeOperationError(phase, node, step, hint string, err error) error {
	if err == nil {
		return nil
	}

	return &resizeOperationError{
		phase: phase,
		node:  node,
		step:  step,
		hint:  hint,
		err:   err,
	}
}

func newResizeControlPlaneResponse(resourceID, vmSize string, deallocateVM bool) *adminapi.ResizeControlPlaneResponse {
	return &adminapi.ResizeControlPlaneResponse{
		ResourceID:   resourceID,
		VMSize:       vmSize,
		DeallocateVM: deallocateVM,
	}
}

func runResizePhase(report *adminapi.ResizeControlPlaneResponse, name string, fn func(*adminapi.ResizeControlPlanePhase) error) error {
	phase := adminapi.ResizeControlPlanePhase{
		Name:   name,
		Status: adminapi.ResizeControlPlaneOperationStatusSucceeded,
	}

	start := time.Now()
	err := fn(&phase)
	phase.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		phase.Status = adminapi.ResizeControlPlaneOperationStatusFailed
		if phase.Message == "" {
			phase.Message = resizeDiagnosticMessage(err)
		}
	}

	if report != nil {
		report.Phases = append(report.Phases, phase)
	}

	return err
}

func buildResizeControlPlaneCloudError(report *adminapi.ResizeControlPlaneResponse, err error) error {
	if err == nil {
		return nil
	}

	elapsedMS := int64(0)
	if report != nil {
		elapsedMS = report.DurationMS
	}

	statusCode := http.StatusInternalServerError
	errorCode := api.CloudErrorCodeInternalServerError
	underlyingMessage := resizeDiagnosticMessage(err)

	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) && cloudErr.CloudErrorBody != nil {
		statusCode = cloudErr.StatusCode
		if cloudErr.Code != "" {
			errorCode = cloudErr.Code
		}
		underlyingMessage = cloudErr.Message
	}

	var opErr *resizeOperationError
	errors.As(err, &opErr)

	target := resizeErrorTarget(opErr)
	details := []api.CloudErrorBody{}
	if report != nil {
		details = append(details, api.CloudErrorBody{
			Code:    "ResizeRequest",
			Target:  report.ResourceID,
			Message: fmt.Sprintf("Requested VM size %s with deallocateVM=%t. Elapsed time before failure: %s.", report.VMSize, report.DeallocateVM, formatResizeDurationMS(elapsedMS)),
		})
	}

	details = append(details, api.CloudErrorBody{
		Code:    errorCode,
		Target:  target,
		Message: underlyingMessage,
	})
	details = append(details, resizeResponseAsCloudErrorDetails(report)...)
	if opErr != nil && opErr.hint != "" {
		details = append(details, api.CloudErrorBody{
			Code:    "InvestigationHint",
			Target:  target,
			Message: opErr.hint,
		})
	}

	return &api.CloudError{
		StatusCode: statusCode,
		CloudErrorBody: &api.CloudErrorBody{
			Code:    errorCode,
			Target:  target,
			Message: fmt.Sprintf("Control plane resize failed during %s after %s. %s", resizeFailureDescription(opErr), formatResizeDurationMS(elapsedMS), underlyingMessage),
			Details: details,
		},
	}
}

func resizeResponseAsCloudErrorDetails(report *adminapi.ResizeControlPlaneResponse) []api.CloudErrorBody {
	if report == nil {
		return nil
	}

	details := make([]api.CloudErrorBody, 0, len(report.Phases)+len(report.Nodes))
	for _, phase := range report.Phases {
		phaseDetail := api.CloudErrorBody{
			Code:    "ResizePhase",
			Target:  phase.Name,
			Message: fmt.Sprintf("%s in %s. %s", strings.ToLower(string(phase.Status)), formatResizeDurationMS(phase.DurationMS), phase.Message),
		}
		for _, check := range phase.Checks {
			phaseDetail.Details = append(phaseDetail.Details, api.CloudErrorBody{
				Code:    "ResizeValidationCheck",
				Target:  check.Name,
				Message: fmt.Sprintf("%s in %s. %s", strings.ToLower(string(check.Status)), formatResizeDurationMS(check.DurationMS), check.Message),
			})
		}
		details = append(details, phaseDetail)
	}

	for _, node := range report.Nodes {
		nodeDetail := api.CloudErrorBody{
			Code:   "ResizeNode",
			Target: node.Name,
			Message: fmt.Sprintf("%s in %s. %s Source VM size: %s. Target VM size: %s.",
				strings.ToLower(string(node.Status)),
				formatResizeDurationMS(node.DurationMS),
				node.Message,
				node.SourceVMSize,
				node.TargetVMSize,
			),
		}
		for _, step := range node.Steps {
			nodeDetail.Details = append(nodeDetail.Details, api.CloudErrorBody{
				Code:    "ResizeNodeStep",
				Target:  fmt.Sprintf("%s/%s", node.Name, step.Name),
				Message: fmt.Sprintf("%s in %s. %s", strings.ToLower(string(step.Status)), formatResizeDurationMS(step.DurationMS), step.Message),
			})
		}
		details = append(details, nodeDetail)
	}

	return details
}

func resizeFailureDescription(opErr *resizeOperationError) string {
	if opErr == nil {
		return "the resize operation"
	}

	if opErr.step != "" && opErr.node != "" {
		return fmt.Sprintf("step %q for node %q", opErr.step, opErr.node)
	}
	if opErr.node != "" {
		return fmt.Sprintf("node %q", opErr.node)
	}
	if opErr.phase != "" {
		return fmt.Sprintf("phase %q", opErr.phase)
	}

	return "the resize operation"
}

func resizeErrorTarget(opErr *resizeOperationError) string {
	if opErr == nil {
		return "resizecontrolplane"
	}

	if opErr.step != "" && opErr.node != "" {
		return fmt.Sprintf("%s/%s", opErr.node, opErr.step)
	}
	if opErr.node != "" {
		return opErr.node
	}
	if opErr.phase != "" {
		return opErr.phase
	}

	return "resizecontrolplane"
}

func resizeDiagnosticMessage(err error) string {
	if err == nil {
		return ""
	}

	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) && cloudErr.CloudErrorBody != nil && cloudErr.Message != "" {
		return cloudErr.Message
	}

	return err.Error()
}

func formatResizeDuration(duration time.Duration) string {
	if duration < time.Millisecond {
		return "<1ms"
	}

	return duration.Round(time.Millisecond).String()
}

func formatResizeDurationMS(durationMS int64) string {
	return formatResizeDuration(time.Duration(durationMS) * time.Millisecond)
}
