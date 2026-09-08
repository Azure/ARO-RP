package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/clusteroperators"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// getPreResizeControlPlaneVMsValidation is the HTTP handler; the underscore
// method below decouples HTTP parsing from logic for testability.
func (f *frontend) getPreResizeControlPlaneVMsValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Strip trailing segment (e.g. "/preresizevalidation") to match the admin resourceID format.
	r.URL.Path = filepath.Dir(r.URL.Path)

	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	desiredVMSize := r.URL.Query().Get("vmSize")

	err := f._getPreResizeControlPlaneVMsValidation(ctx, resType, resName, resGroupName, resourceID, desiredVMSize, log)

	adminReply(log, w, nil, nil, err)
}

// _getPreResizeControlPlaneVMsValidation runs all pre-flight checks before
// the ResizeControlPlaneVMs orchestration loop starts. Failing early prevents
// leaving the cluster degraded with reduced etcd quorum.
func (f *frontend) _getPreResizeControlPlaneVMsValidation(
	ctx context.Context,
	resType, resName, resGroupName, resourceID, desiredVMSize string,
	log *logrus.Entry,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf(
				"The Resource '%s/%s' under resource group '%s' was not found.",
				resType, resName, resGroupName,
			))
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

	return f.preResizeControlPlaneVMsValidation(ctx, doc, subscriptionDoc, k, a, desiredVMSize, log)
}

func (f *frontend) preResizeControlPlaneVMsValidation(
	ctx context.Context,
	doc *api.OpenShiftClusterDocument,
	subscriptionDoc *api.SubscriptionDocument,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
	desiredVMSize string,
	log *logrus.Entry,
) error {
	preValidator := &preResizeValidator{
		doc:             doc,
		log:             log,
		subscriptionDoc: subscriptionDoc,
		k:               k,
		a:               a,
		desiredVMSize:   desiredVMSize,
	}
	validationSteps := []steps.Step{
		steps.Action(preValidator.validateAPIServerReadyz),
		steps.Action(preValidator.validateVMSKU),
		steps.Concurrent("preresize", []steps.Step{
			steps.Action(preValidator.validateAPIServerHealth),
			steps.Action(preValidator.validateAPIServerPods),
			steps.Action(preValidator.validateEtcdHealth),
			steps.Action(preValidator.validateClusterSP),
			steps.Action(preValidator.validateCPMSNotActive),
		}),
		steps.Action(preValidator.validateResizeControlPlaneInventory),
	}

	_, err := steps.RunWithWrappedError(ctx, log, time.Second*10, validationSteps, time.Now)

	return err
}

// preResizeValidator wraps the validation functions and stores the clients and
// request metadata. This makes it possible to let the individual functions
// only take context.Context as their single argument. This is needed so we can
// use them as argument to steps.Action(...).
type preResizeValidator struct {
	doc             *api.OpenShiftClusterDocument
	log             *logrus.Entry
	subscriptionDoc *api.SubscriptionDocument
	k               adminactions.KubeActions
	a               adminactions.AzureActions
	desiredVMSize   string
}

func (v *preResizeValidator) validateAPIServerReadyz(ctx context.Context) error {
	return v.k.CheckAPIServerReadyz(ctx)
}

func (v *preResizeValidator) validateVMSKU(ctx context.Context) error {
	return validateVMSKU(ctx, v.doc, v.subscriptionDoc, v.desiredVMSize, v.k, v.a)
}

func (v *preResizeValidator) validateResizeControlPlaneInventory(ctx context.Context) error {
	return validateLiveControlPlaneInventory(v.log, ctx, v.k, v.a, v.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
}

// checkResizeComputeQuota verifies that the subscription has enough remaining
// compute quota (both per-family and overall regional "cores") to resize all
// master nodes.
//
// Unlike validateQuota in quota_validation.go (which checks absolute totals for
// cluster creation), this computes the incremental delta per VM. Each master VM
// may have a different current size (e.g. after a partial resize), so we
// calculate the delta individually and sum across all VMs that need resizing.
//
// Same-family resizes only need (newCores − currentCores) per VM; cross-family
// resizes need the full new cores for the target family but only the net delta
// for regional "cores".
//
// This checks subscription-level quota only, not Azure regional datacenter
// capacity — without a capacity reservation, AllocationFailed errors can only
// be detected at ARM PUT time.
func checkResizeComputeQuota(ctx context.Context, a adminactions.AzureActions, location string, currentVMSizes []string, desiredVMSize string) error {
	newSizeStruct, ok := validate.VMSizeFromName(api.VMSize(desiredVMSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The provided VM SKU '%s' is not supported.", desiredVMSize))
	}

	requiredByQuota := map[string]int{}

	for _, currentVMSize := range currentVMSizes {
		if strings.EqualFold(currentVMSize, desiredVMSize) {
			continue // VM already at desired size, no quota needed
		}

		currentSizeStruct, ok := validate.VMSizeFromName(api.VMSize(currentVMSize))
		if !ok {
			return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "currentVMSize",
				fmt.Sprintf("The current VM SKU '%s' could not be resolved.", currentVMSize))
		}

		// Same family: only the delta matters. Cross-family: full new cores needed.
		additionalFamilyCores := newSizeStruct.CoreCount
		if newSizeStruct.Family == currentSizeStruct.Family {
			additionalFamilyCores = newSizeStruct.CoreCount - currentSizeStruct.CoreCount
		}

		if additionalFamilyCores > 0 {
			requiredByQuota[newSizeStruct.Family] += additionalFamilyCores
		}

		// Regional "cores" delta accounts for freed cores from the old VM.
		regionalDelta := newSizeStruct.CoreCount - currentSizeStruct.CoreCount
		if regionalDelta > 0 {
			requiredByQuota["cores"] += regionalDelta
		}
	}

	// All VMs already at desired size or downsizing — no quota check needed.
	if len(requiredByQuota) == 0 {
		return nil
	}

	usages, err := a.ListComputeUsage(ctx, location)
	if err != nil {
		return err
	}

	for _, usage := range usages {
		if usage.Name == nil || usage.Name.Value == nil {
			continue
		}
		required, ok := requiredByQuota[*usage.Name.Value]
		if !ok || required <= 0 {
			continue
		}
		if usage.Limit == nil || usage.CurrentValue == nil {
			continue
		}
		remaining := *usage.Limit - int64(*usage.CurrentValue)
		if int64(required) > remaining {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "vmSize",
				fmt.Sprintf("Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.",
					*usage.Name.Value, *usage.Limit, *usage.CurrentValue, required))
		}
	}

	// If a quota entry is not in the usage list, assume no limit applies.
	return nil
}

// validateCPMSNotActive verifies that the ControlPlaneMachineSet is not Active.
// If it is active, direct VM manipulation would conflict with the CPMS operator.
// Only NotFound / CRD-not-installed errors are treated as "CPMS absent";
// all other errors fail the operation closed so we don't bypass the safety check.
func (v *preResizeValidator) validateCPMSNotActive(ctx context.Context) error {
	rawCPMS, err := v.k.KubeGet(ctx, "ControlPlaneMachineSet.machine.openshift.io", machineNamespace, "cluster")
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

// validateAPIServerHealth verifies that the kube-apiserver ClusterOperator is healthy
// (Available=True, Progressing=False, Degraded=False).
// Note: API server reachability is checked earlier via CheckAPIServerReadyz
func (v *preResizeValidator) validateAPIServerHealth(ctx context.Context) error {
	rawCO, err := v.k.KubeGet(ctx, "ClusterOperator.config.openshift.io", "", "kube-apiserver")
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver",
			fmt.Sprintf("Failed to retrieve kube-apiserver ClusterOperator: %v", err),
		)
	}

	var co configv1.ClusterOperator
	if err := json.Unmarshal(rawCO, &co); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver",
			fmt.Sprintf("Failed to parse kube-apiserver ClusterOperator: %v", err),
		)
	}

	if !clusteroperators.IsOperatorAvailable(&co) {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "kube-apiserver",
			fmt.Sprintf("kube-apiserver is not healthy: %s. Resize is not safe while the API server is degraded.",
				clusteroperators.OperatorStatusText(&co)),
		)
	}

	return nil
}

func (v *preResizeValidator) validateAPIServerPods(ctx context.Context) error {
	const (
		kubeAPIServerNamespace     = "openshift-kube-apiserver"
		kubeAPIServerLabelSelector = "app=openshift-kube-apiserver"
	)

	rawPods, err := v.k.KubeList(ctx, "Pod", kubeAPIServerNamespace, kubeAPIServerLabelSelector)
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver-pods",
			fmt.Sprintf("Failed to list pods in %s namespace: %v", kubeAPIServerNamespace, err),
		)
	}

	var podList corev1.PodList
	if err := json.Unmarshal(rawPods, &podList); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver-pods",
			fmt.Sprintf("Failed to parse pod list: %v", err),
		)
	}

	var unhealthyPods []string
	for _, pod := range podList.Items {
		if err := validatePodHealth(&pod); err != nil {
			unhealthyPods = append(unhealthyPods, fmt.Sprintf("%s (%s)", pod.Name, err.Error()))
		}
	}

	apiServerPodCount := len(podList.Items)

	if apiServerPodCount != api.ControlPlaneNodeCount {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "kube-apiserver-pods",
			fmt.Sprintf("Expected %d kube-apiserver pods, found %d. Resize is not safe without full API server redundancy.",
				api.ControlPlaneNodeCount, apiServerPodCount),
		)
	}

	if len(unhealthyPods) > 0 {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "kube-apiserver-pods",
			fmt.Sprintf("Unhealthy kube-apiserver pods: %v. Resize is not safe without full API server redundancy.",
				unhealthyPods),
		)
	}

	return nil
}

func validatePodHealth(pod *corev1.Pod) error {
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("phase: %s", pod.Status.Phase)
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			if cond.Status != corev1.ConditionTrue {
				return fmt.Errorf("not ready")
			}
			return nil
		}
	}

	return fmt.Errorf("ready condition not found")
}

// validateEtcdHealth verifies that the etcd ClusterOperator is healthy.
// Resizing takes a master offline, so all etcd members must be healthy.
func (v *preResizeValidator) validateEtcdHealth(ctx context.Context) error {
	return validateEtcdHealth(ctx, v.k)
}

// validateEtcdHealth verifies that the etcd ClusterOperator is healthy.
// Resizing takes a master offline, so all etcd members must be healthy.
func validateEtcdHealth(ctx context.Context, k adminactions.KubeActions) error {
	rawCO, err := k.KubeGet(ctx, "ClusterOperator.config.openshift.io", "", "etcd")
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "etcd",
			fmt.Sprintf("Failed to retrieve etcd ClusterOperator: %v", err),
		)
	}

	var co configv1.ClusterOperator
	if err := json.Unmarshal(rawCO, &co); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "etcd",
			fmt.Sprintf("Failed to parse etcd ClusterOperator: %v", err),
		)
	}

	if !clusteroperators.IsOperatorAvailable(&co) {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "etcd",
			fmt.Sprintf("etcd is not healthy: %s. Resize is not safe while etcd quorum is at risk.",
				clusteroperators.OperatorStatusText(&co)),
		)
	}

	return nil
}

// validateClusterSP checks the ServicePrincipalValid condition on the ARO
// Cluster CRD. The SP is required for the ARM VM PUT during resize.
func (v *preResizeValidator) validateClusterSP(ctx context.Context) error {
	rawCluster, err := v.k.KubeGet(ctx, "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName)
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "servicePrincipal",
			fmt.Sprintf("Failed to retrieve ARO Cluster resource: %v", err),
		)
	}

	var cluster arov1alpha1.Cluster
	if err := json.Unmarshal(rawCluster, &cluster); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "servicePrincipal",
			fmt.Sprintf("Failed to parse ARO Cluster resource: %v", err),
		)
	}

	for _, cond := range cluster.Status.Conditions {
		if cond.Type == arov1alpha1.ServicePrincipalValid {
			if cond.Status == operatorv1.ConditionTrue {
				return nil
			}
			return api.NewCloudError(
				http.StatusConflict,
				api.CloudErrorCodeInvalidServicePrincipalCredentials, "servicePrincipal",
				fmt.Sprintf("Cluster Service Principal is invalid: %s", cond.Message),
			)
		}
	}

	return api.NewCloudError(
		http.StatusConflict,
		api.CloudErrorCodeInvalidServicePrincipalCredentials, "servicePrincipal",
		"ServicePrincipalValid condition not found on the ARO Cluster resource. The ARO operator may not have reconciled yet.",
	)
}

func validateVMSKU(
	ctx context.Context,
	doc *api.OpenShiftClusterDocument,
	subscriptionDoc *api.SubscriptionDocument,
	desiredVMSize string,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
) error {
	if desiredVMSize == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize", "The provided vmSize is empty.")
	}

	err := validateAdminMasterVMSize(desiredVMSize)
	if err != nil {
		return err
	}

	filteredSkus, err := a.VMGetSKUs(ctx, []string{desiredVMSize})
	if err != nil {
		return err
	}

	location := doc.OpenShiftCluster.Location
	sku, err := checkSKUAvailability(filteredSkus, location, "vmSize", desiredVMSize)
	if err != nil {
		return err
	}

	err = checkSKURestriction(sku, location, "vmSize")
	if err != nil {
		return err
	}

	currentVMSizes, err := currentControlPlaneVMSizes(ctx, doc, k, a)
	if err != nil {
		return err
	}

	err = checkResizeComputeQuota(ctx, a, location, currentVMSizes, desiredVMSize)
	if err != nil {
		return err
	}

	return nil
}

func currentControlPlaneVMSizes(
	ctx context.Context,
	doc *api.OpenShiftClusterDocument,
	k adminactions.KubeActions,
	a adminactions.AzureActions,
) ([]string, error) {
	machines, err := getControlPlaneMachines(ctx, k)
	if err != nil {
		return nil, err
	}

	if len(machines) != api.ControlPlaneNodeCount {
		return nil, api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError,
			"controlPlaneMachines",
			fmt.Sprintf(
				"Expected %d control plane machines but found %d. Resize cannot proceed until all control plane machines are present.",
				api.ControlPlaneNodeCount,
				len(machines),
			),
		)
	}

	clusterRGName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	machineNames := slices.Sorted(maps.Keys(machines))
	sizes := make([]string, 0, len(machineNames))

	for _, machineName := range machineNames {
		vm, err := a.GetVirtualMachine(ctx, clusterRGName, machineName, "")
		if err != nil {
			return nil, api.NewCloudError(
				http.StatusInternalServerError,
				api.CloudErrorCodeInternalServerError,
				fmt.Sprintf("controlPlaneVM/%s", machineName),
				fmt.Sprintf("Failed to retrieve current control plane VM %q from Azure: %v", machineName, err),
			)
		}

		if vm.VirtualMachineProperties == nil || vm.HardwareProfile == nil {
			return nil, api.NewCloudError(
				http.StatusInternalServerError,
				api.CloudErrorCodeInternalServerError,
				fmt.Sprintf("controlPlaneVM/%s", machineName),
				fmt.Sprintf("Control plane VM %q has no HardwareProfile in Azure. Resize cannot proceed until all control plane VM details are available.", machineName),
			)
		}

		sizes = append(sizes, string(vm.HardwareProfile.VMSize))
	}

	return sizes, nil
}
