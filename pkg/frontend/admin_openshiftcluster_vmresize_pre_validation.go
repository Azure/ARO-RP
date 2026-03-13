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

// getPreResizeControlPlaneVMsValidation is the HTTP handler; the underscore
// method below decouples HTTP parsing from logic for testability.
func (f *frontend) getPreResizeControlPlaneVMsValidation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Strip trailing segment (e.g. "/preresizevalidation") to match the admin resourceID format.
	r.URL.Path = filepath.Dir(r.URL.Path)

	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	vmSize := r.URL.Query().Get("vmSize")

	b, err := f._getPreResizeControlPlaneVMsValidation(ctx, resType, resName, resGroupName, resourceID, vmSize, log)

	adminReply(log, w, nil, b, err)
}

// _getPreResizeControlPlaneVMsValidation runs all pre-flight checks before
// the ResizeControlPlaneVMs orchestration loop starts. Failing early prevents
// leaving the cluster degraded with reduced etcd quorum.
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

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	// Run checks in parallel, collecting all errors so the caller sees every failure at once.
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
	wg.Go(func() { collect(validateAPIServerHealth(ctx, k)) })
	wg.Go(func() { collect(validateEtcdHealth(ctx, k)) })
	wg.Go(func() { collect(validateClusterSP(ctx, k)) })

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

// defaultValidateResizeQuota creates an FP-authorized compute usage client and
// delegates to checkResizeComputeQuota. Injected via f.validateResizeQuota so
// tests can swap it with quotaCheckDisabled.
func defaultValidateResizeQuota(ctx context.Context, environment env.Interface, subscriptionDoc *api.SubscriptionDocument, location, currentVMSize, vmSize string) error {
	tenantID := subscriptionDoc.Subscription.Properties.TenantID

	fpAuthorizer, err := environment.FPAuthorizer(tenantID, nil, environment.Environment().ResourceManagerScope)
	if err != nil {
		return err
	}

	spComputeUsage := compute.NewUsageClient(environment.Environment(), subscriptionDoc.ID, fpAuthorizer)
	return checkResizeComputeQuota(ctx, spComputeUsage, location, currentVMSize, vmSize)
}

// checkResizeComputeQuota verifies that the subscription has enough remaining
// compute quota in the target VM family to resize all master nodes.
//
// Unlike validateQuota in quota_validation.go (which checks absolute totals for
// cluster creation), this computes the incremental delta: same-family resizes
// only need (newCores − currentCores) × nodeCount; cross-family resizes need
// the full new cores for the target family.
//
// This checks subscription-level quota only, not Azure regional datacenter
// capacity — without a capacity reservation, AllocationFailed errors can only
// be detected at ARM PUT time.
func checkResizeComputeQuota(ctx context.Context, spComputeUsage compute.UsageClient, location, currentVMSize, vmSize string) error {
	newSizeStruct, ok := validate.VMSizeFromName(api.VMSize(vmSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The provided VM SKU '%s' is not supported.", vmSize))
	}

	currentSizeStruct, ok := validate.VMSizeFromName(api.VMSize(currentVMSize))
	if !ok {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "vmSize",
			fmt.Sprintf("The current VM SKU '%s' could not be resolved.", currentVMSize))
	}

	// Same family: only the delta matters. Cross-family: full new cores needed.
	additionalCoresPerNode := newSizeStruct.CoreCount
	if newSizeStruct.Family == currentSizeStruct.Family {
		additionalCoresPerNode = newSizeStruct.CoreCount - currentSizeStruct.CoreCount
		if additionalCoresPerNode <= 0 {
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

	// If the family is not in the usage list, assume no limit applies.
	return nil
}

// quotaCheckDisabled is a no-op replacement for f.validateResizeQuota in tests.
func quotaCheckDisabled(_ context.Context, _ env.Interface, _ *api.SubscriptionDocument, _, _, _ string) error {
	return nil
}

// validateAPIServerHealth verifies that the kube-apiserver ClusterOperator is
// healthy (Available=True, Progressing=False, Degraded=False).
func validateAPIServerHealth(ctx context.Context, k adminactions.KubeActions) error {
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

// validateEtcdHealth verifies that the etcd ClusterOperator is healthy.
// Resizing takes a master offline, so all etcd members must be healthy.
func validateEtcdHealth(ctx context.Context, k adminactions.KubeActions) error {
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

// validateClusterSP checks the ServicePrincipalValid condition on the ARO
// Cluster CRD. The SP is required for the ARM VM PUT during resize.
func validateClusterSP(ctx context.Context, k adminactions.KubeActions) error {
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

	return api.NewCloudError(
		http.StatusConflict,
		api.CloudErrorCodeInvalidServicePrincipalCredentials, "servicePrincipal",
		"ServicePrincipalValid condition not found on the ARO Cluster resource. The ARO operator may not have reconciled yet.")
}

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

	err := validateAdminMasterVMSize(vmSize)
	if err != nil {
		return err
	}

	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return err
	}

	skus, err := a.VMSizeList(ctx)
	if err != nil {
		return err
	}

	location := doc.OpenShiftCluster.Location

	filteredSkus := computeskus.FilterVMSizes(skus, location)

	sku, err := checkSKUAvailability(filteredSkus, location, "vmSize", vmSize)
	if err != nil {
		return err
	}

	err = checkSKURestriction(sku, location, "vmSize")
	if err != nil {
		return err
	}

	currentVMSize := string(doc.OpenShiftCluster.Properties.MasterProfile.VMSize)
	err = f.validateResizeQuota(ctx, f.env, subscriptionDoc, location, currentVMSize, vmSize)
	if err != nil {
		return err
	}

	return nil
}
