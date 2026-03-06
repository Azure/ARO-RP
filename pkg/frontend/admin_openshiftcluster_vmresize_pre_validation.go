package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
)

type preResizeValidationResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// getPreResizeControlPlaneVMsValidation is the HTTP handler that decouples URL
// parameter extraction from business logic.  The underscore function below can
// be invoked directly by other Go packages without mocking an HTTP request.
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
//
// The checks below are intentionally structured as independent validations so
// that future iterations can execute them in parallel goroutines (e.g. using
// errgroup) without interference—each validation path reads its own data and
// produces its own error.
func (f *frontend) _getPreResizeControlPlaneVMsValidation(ctx context.Context, resType, resName, resGroupName, resourceID, vmSize string, log *logrus.Entry) ([]byte, error) {
	if vmSize == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize", "The provided vmSize is empty.")
	}

	// Reject early if the requested size is not in the ARO-supported master VM
	// sizes list, before making any Azure API calls.
	err := validateAdminMasterVMSize(vmSize)
	if err != nil {
		return nil, err
	}

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

	// TODO: Validate Cluster Service Principal validity.
	// The SP is required for the implicit ARM VM PUT during resize.  If it is
	// expired or lacks permissions, ARM will reject the call and the node stays
	// offline.  Use kubeActionsFactory to query the aro-operator's
	// serviceprincipalchecker condition (decoupled path, parallelizable).

	// TODO: Validate API server health.
	// The kube-apiserver must be Available=True and Degraded=False before we
	// start cordoning nodes.  Use kubeActionsFactory to fetch the
	// kube-apiserver ClusterOperator status (decoupled path, parallelizable).

	// --- SKU availability and quota validation (Azure Compute queries) ---

	// AzureActions wraps FP-authenticated clients scoped to the cluster's
	// subscription and resource group.
	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	// VMSizeList queries the Azure Compute Resource SKUs API filtered by the RP
	// region.  The raw list includes all resource types, not just VMs.
	skus, err := a.VMSizeList(ctx)
	if err != nil {
		return nil, err
	}

	location := doc.OpenShiftCluster.Location

	// FilterVMSizes narrows the raw SKU list to virtualMachines in the cluster's
	// region and returns a map keyed by SKU name for O(1) lookups.
	filteredSkus := computeskus.FilterVMSizes(skus, location)

	// Verify the target SKU actually exists in this region—zone restrictions or
	// region-specific unavailability would cause the ARM PUT to fail.
	sku, err := checkSKUAvailability(filteredSkus, location, "vmSize", vmSize)
	if err != nil {
		return nil, err
	}

	// Restrictions are subscription-specific (e.g. enterprise agreement
	// limitations, policy-based blocks).  A restricted SKU would silently fail
	// during the resize ARM call.
	err = checkSKURestriction(sku, location, "vmSize")
	if err != nil {
		return nil, err
	}

	// Verify the subscription has enough remaining cores in the target VM
	// family.  The resize operation stops the old VM first (releasing its
	// cores), so we conservatively check for one node's worth of new cores.
	err = f.validateResizeQuota(ctx, f.env, subscriptionDoc, location, vmSize)
	if err != nil {
		return nil, err
	}

	result := preResizeValidationResult{
		Status:  "success",
		Message: fmt.Sprintf("SKU '%s' is available and valid for control plane resize in region '%s'.", vmSize, location),
	}

	return json.Marshal(result)
}

// defaultValidateResizeQuota creates an FP-authorized compute usage client
// scoped to the customer's subscription and delegates to checkResizeComputeQuota.
// Injected via f.validateResizeQuota so tests can swap it with quotaCheckDisabled.
func defaultValidateResizeQuota(ctx context.Context, environment env.Interface, subscriptionDoc *api.SubscriptionDocument, location, vmSize string) error {
	tenantID := subscriptionDoc.Subscription.Properties.TenantID

	// FPAuthorizer authenticates as the RP's first-party identity in the
	// customer's tenant, which has reader access to compute usage.
	fpAuthorizer, err := environment.FPAuthorizer(tenantID, nil, environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	spComputeUsage := compute.NewUsageClient(environment.Environment(), subscriptionDoc.ID, fpAuthorizer)
	return checkResizeComputeQuota(ctx, spComputeUsage, location, vmSize)
}

// checkResizeComputeQuota verifies that the subscription has enough remaining
// compute quota in the target VM family for at least one instance of the
// requested size.  During resize the old VM is stopped first (releasing its
// cores), so a single node's worth of new cores is the conservative
// requirement.  This is a pure function for direct unit testing.
func checkResizeComputeQuota(ctx context.Context, spComputeUsage compute.UsageClient, location, vmSize string) error {
	// Resolve the VM size name to its family and core count so we know which
	// quota counter to check and how many cores are needed.
	vmSizeStruct, ok := validate.VMSizeFromName(api.VMSize(vmSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The provided VM SKU '%s' is not supported.", vmSize))
	}

	usages, err := spComputeUsage.List(ctx, location)
	if err != nil {
		return err
	}

	for _, usage := range usages {
		if usage.Name == nil || usage.Name.Value == nil {
			continue
		}
		if *usage.Name.Value == vmSizeStruct.Family {
			remaining := *usage.Limit - int64(*usage.CurrentValue)
			if int64(vmSizeStruct.CoreCount) > remaining {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeResourceQuotaExceeded, "vmSize",
					fmt.Sprintf("Resource quota of %s exceeded. Maximum allowed: %d, Current in use: %d, Additional requested: %d.",
						vmSizeStruct.Family, *usage.Limit, *usage.CurrentValue, vmSizeStruct.CoreCount))
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
func quotaCheckDisabled(_ context.Context, _ env.Interface, _ *api.SubscriptionDocument, _, _ string) error {
	return nil
}
