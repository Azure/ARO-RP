package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/clusteroperators"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

// getPreResizeControlPlaneVMsValidation is the HTTP handler that decouples URL
// parameter extraction from business logic. The underscore method below
// decouples HTTP parsing from logic so it can be invoked directly by internal
// callers (for example, tests) without mocking an HTTP request.
func (f *frontend) getPreResizeControlPlaneVMsValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Strip the trailing path segment (e.g. "/preresizevalidation") so that
	// r.URL.Path ends at the resource name, matching the admin resourceID format.
	r.URL.Path = filepath.Dir(r.URL.Path)

	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	vmSize := r.URL.Query().Get("vmSize")

	b, err := f._getPreResizeControlPlaneVMsValidation(ctx, resType, resName, resGroupName, resourceID, vmSize, log)

	adminReply(log, w, nil, b, err)
}

// _getPreResizeControlPlaneVMsValidation runs all pre-flight checks that must
// pass before the Geneva Action's ResizeControlPlaneVMs orchestration loop is
// allowed to cordon/drain/stop any master node.  Failing early here prevents
// leaving the cluster in a degraded state with reduced etcd quorum.
func (f *frontend) _getPreResizeControlPlaneVMsValidation(
	ctx context.Context,
	resType, resName, resGroupName, resourceID, vmSize string,
	log *logrus.Entry,
) ([]byte, error) {
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf(
				"The Resource '%s/%s' under resource group '%s' was not found.",
				resType, resName, resGroupName))
	case err != nil:
		return nil, err
	}

	// Subscription doc carries the tenant ID needed to authenticate against the
	// customer's Azure subscription for SKU and quota queries.
	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	// Create kubeActions once, shared by API server and SP checks.
	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	// Run all pre-flight checks in parallel.  Errors are collected via mutex
	// so that all checks run to completion and the caller sees every failure
	// at once, rather than only the first one.
	var (
		mu      sync.Mutex
		details []api.CloudErrorBody
	)
	collect := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var ce *api.CloudError
		if errors.As(err, &ce) && ce.CloudErrorBody != nil {
			details = append(details, *ce.CloudErrorBody)
		} else {
			details = append(details, api.CloudErrorBody{
				Code:    api.CloudErrorCodeInternalServerError,
				Message: err.Error(),
			})
		}
	}

	var wg sync.WaitGroup

	wg.Go(func() { collect(f.validateVMSKU(ctx, doc, subscriptionDoc, vmSize, log)) })
	wg.Go(func() { collect(f.validateAPIServerHealth(ctx, k)) })
	wg.Go(func() { collect(f.validateEtcdHealth(ctx, k)) })
	wg.Go(func() { collect(f.validateClusterSP(ctx, k)) })

	wg.Wait()

	if len(details) > 0 {
		return nil, &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeInvalidParameter,
				Message: "Pre-flight validation failed.",
				Details: details,
			},
		}
	}

	return json.Marshal("All pre-flight checks passed")
}

// defaultValidateResizeQuota creates an FP-authorized compute usage client
// scoped to the customer's subscription and delegates to checkResizeComputeQuota.
// Injected via f.validateResizeQuota so tests can swap it with quotaCheckDisabled.
func defaultValidateResizeQuota(ctx context.Context, environment env.Interface, subscriptionDoc *api.SubscriptionDocument, location, currentVMSize, vmSize string) error {
	tenantID := subscriptionDoc.Subscription.Properties.TenantID

	// FPAuthorizer authenticates as the RP's first-party identity in the
	// customer's tenant, which has reader access to compute usage.
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, nil, environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	spComputeUsage := compute.NewUsageClient(environment.Environment(), subscriptionDoc.ID, fpAuthorizer)
	return checkResizeComputeQuota(ctx, spComputeUsage, location, currentVMSize, vmSize)
}

// checkResizeComputeQuota verifies that the subscription has enough remaining
// compute quota in the target VM family to resize all master nodes.  The resize
// operation processes nodes sequentially (stop → resize → start), so after all
// nodes are processed the total usage change is api.ControlPlaneNodeCount × delta.
//
//   - If the current and new VMs share the same family, only the per-node delta
//     (newCores − currentCores) matters, multiplied by the number of masters.
//     If the new size is equal or smaller, no additional quota is needed.
//   - If the families differ, the full new core count × api.ControlPlaneNodeCount is
//     required because stopping the old VM frees quota in a different family.
//
// NOTE: This checks subscription-level quota only, not Azure regional
// datacenter capacity.  Capacity reservations can guarantee hardware
// availability in a region (see https://learn.microsoft.com/en-us/azure/virtual-machines/capacity-reservation-overview),
// but this validation does not account for them.  Without a reservation,
// AllocationFailed errors can only be detected at ARM PUT time.
func checkResizeComputeQuota(ctx context.Context, spComputeUsage compute.UsageClient, location, currentVMSize, vmSize string) error {
	// Resolve the new VM size name to its family and core count.
	newSizeStruct, ok := validate.VMSizeFromName(api.VMSize(vmSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The provided VM SKU '%s' is not supported.", vmSize))
	}

	// Resolve the current VM size to determine how many cores will be freed.
	currentSizeStruct, ok := validate.VMSizeFromName(api.VMSize(currentVMSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The current VM SKU '%s' could not be resolved.", currentVMSize))
	}

	// Compute the per-node core delta.  When both sizes belong to the same
	// family, the old VM's cores are freed before the new VM is created, so
	// only the delta matters.  Multiply by api.ControlPlaneNodeCount because all master
	// nodes are resized sequentially and the peak quota is at the final state.
	additionalCoresPerNode := newSizeStruct.CoreCount
	if newSizeStruct.Family == currentSizeStruct.Family {
		additionalCoresPerNode = newSizeStruct.CoreCount - currentSizeStruct.CoreCount
		if additionalCoresPerNode <= 0 {
			// Downsizing or same size within the same family — no extra quota needed.
			return nil
		}
	}
	totalAdditionalCores := additionalCoresPerNode * api.ControlPlaneNodeCount

	usages, err := spComputeUsage.List(ctx, location)
	if err != nil {
		return err
	}

	for _, usage := range usages {
		if usage.Name == nil || usage.Name.Value == nil {
			continue
		}
		if *usage.Name.Value == newSizeStruct.Family {
			if usage.Limit == nil || usage.CurrentValue == nil {
				continue
			}
			remaining := *usage.Limit - int64(*usage.CurrentValue)
			if int64(totalAdditionalCores) > remaining {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "vmSize",
					fmt.Sprintf("Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.",
						newSizeStruct.Family, *usage.Limit, *usage.CurrentValue, totalAdditionalCores))
			}
			return nil
		}
	}

	// If the family is not in the usage list, assume no quota limit applies.
	// This matches the existing validateQuota behavior—the Usage API may omit
	// families that have no enforced cap.
	return nil
}

// quotaCheckDisabled is a no-op replacement for f.validateResizeQuota in
// integration tests, avoiding the need to create real FP-authorized Azure
// clients.
func quotaCheckDisabled(_ context.Context, _ env.Interface, _ *api.SubscriptionDocument, _, _, _ string) error {
	return nil
}

// validateAPIServerHealth queries the kube-apiserver ClusterOperator via the
// cluster's Kubernetes API and verifies that it is healthy (Available=True,
// Progressing=False, Degraded=False).
func (f *frontend) validateAPIServerHealth(ctx context.Context, k adminactions.KubeActions) error {
	rawCO, err := k.KubeGet(ctx, "ClusterOperator.config.openshift.io", "", "kube-apiserver")
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver",
			fmt.Sprintf("Failed to retrieve kube-apiserver ClusterOperator: %v", err))
	}

	var co configv1.ClusterOperator
	if err := json.Unmarshal(rawCO, &co); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "kube-apiserver",
			fmt.Sprintf("Failed to parse kube-apiserver ClusterOperator: %v", err))
	}

	if !clusteroperators.IsOperatorAvailable(&co) {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "kube-apiserver",
			fmt.Sprintf("kube-apiserver is not healthy: %s. Resize is not safe while the API server is degraded.",
				clusteroperators.OperatorStatusText(&co)))
	}

	return nil
}

// validateEtcdHealth queries the etcd ClusterOperator and verifies that it is
// healthy (Available=True, Progressing=False, Degraded=False).  Etcd quorum
// requires at least 2 of 3 members; resizing a master takes a node offline, so
// all members must be healthy before proceeding.
func (f *frontend) validateEtcdHealth(ctx context.Context, k adminactions.KubeActions) error {
	rawCO, err := k.KubeGet(ctx, "ClusterOperator.config.openshift.io", "", "etcd")
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "etcd",
			fmt.Sprintf("Failed to retrieve etcd ClusterOperator: %v", err))
	}

	var co configv1.ClusterOperator
	if err := json.Unmarshal(rawCO, &co); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "etcd",
			fmt.Sprintf("Failed to parse etcd ClusterOperator: %v", err))
	}

	if !clusteroperators.IsOperatorAvailable(&co) {
		return api.NewCloudError(
			http.StatusConflict,
			api.CloudErrorCodeRequestNotAllowed, "etcd",
			fmt.Sprintf("etcd is not healthy: %s. Resize is not safe while etcd quorum is at risk.",
				clusteroperators.OperatorStatusText(&co)))
	}

	return nil
}

// validateClusterSP queries the ARO Cluster CRD to check the ServicePrincipalValid
// condition set by the serviceprincipalchecker operator controller.  The cluster
// Service Principal is required for the implicit ARM VM PUT during resize; if
// it is expired or lacks permissions the resize will fail with the node offline.
func (f *frontend) validateClusterSP(ctx context.Context, k adminactions.KubeActions) error {
	rawCluster, err := k.KubeGet(ctx, "Cluster.aro.openshift.io", "", arov1alpha1.SingletonClusterName)
	if err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "servicePrincipal",
			fmt.Sprintf("Failed to retrieve ARO Cluster resource: %v", err))
	}

	var cluster arov1alpha1.Cluster
	if err := json.Unmarshal(rawCluster, &cluster); err != nil {
		return api.NewCloudError(
			http.StatusInternalServerError,
			api.CloudErrorCodeInternalServerError, "servicePrincipal",
			fmt.Sprintf("Failed to parse ARO Cluster resource: %v", err))
	}

	for _, cond := range cluster.Status.Conditions {
		if cond.Type == arov1alpha1.ServicePrincipalValid {
			if cond.Status == operatorv1.ConditionTrue {
				return nil
			}
			return api.NewCloudError(
				http.StatusConflict,
				api.CloudErrorCodeInvalidServicePrincipalCredentials, "servicePrincipal",
				fmt.Sprintf("Cluster Service Principal is invalid: %s", cond.Message))
		}
	}

	// Condition not found — the checker may not have run yet.
	return api.NewCloudError(
		http.StatusConflict,
		api.CloudErrorCodeInvalidServicePrincipalCredentials, "servicePrincipal",
		"ServicePrincipalValid condition not found on the ARO Cluster resource. The serviceprincipalchecker may not have run yet.")
}

// --- SKU availability and quota validation (Azure Compute queries) ---
func (f *frontend) validateVMSKU(
	ctx context.Context,
	doc *api.OpenShiftClusterDocument,
	subscriptionDoc *api.SubscriptionDocument,
	vmSize string,
	log *logrus.Entry,
) error {
	if vmSize == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize", "The provided vmSize is empty.")
	}

	// Reject early if the requested size is not in the ARO-supported master VM
	// sizes list, before making any Azure API calls.
	err := validateAdminMasterVMSize(vmSize)
	if err != nil {
		return err
	}

	// AzureActions wraps FP-authenticated clients scoped to the cluster's
	// subscription and resource group.
	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return err
	}

	// VMSizeList queries the Azure Compute Resource SKUs API filtered by the RP
	// region.  The raw list includes all resource types, not just VMs.
	skus, err := a.VMSizeList(ctx)
	if err != nil {
		return err
	}

	location := doc.OpenShiftCluster.Location

	// FilterVMSizes narrows the raw SKU list to virtualMachines in the cluster's
	// region and returns a map keyed by SKU name for O(1) lookups.
	filteredSkus := computeskus.FilterVMSizes(skus, location)

	// Verify the target SKU actually exists in this region—zone restrictions or
	// region-specific unavailability would cause the ARM PUT to fail.
	sku, err := checkSKUAvailability(filteredSkus, location, "vmSize", vmSize)
	if err != nil {
		return err
	}

	// Restrictions are subscription-specific (e.g. enterprise agreement
	// limitations, policy-based blocks).  A restricted SKU would silently fail
	// during the resize ARM call.
	err = checkSKURestriction(sku, location, "vmSize")
	if err != nil {
		return err
	}

	// Verify the subscription has enough remaining cores in the target VM
	// family.  The resize operation stops the old VM first (releasing its
	// cores), so we only check the delta when both sizes share a family.
	currentVMSize := string(doc.OpenShiftCluster.Properties.MasterProfile.VMSize)
	err = f.validateResizeQuota(ctx, f.env, subscriptionDoc, location, currentVMSize, vmSize)
	if err != nil {
		return err
	}

	return nil
}
