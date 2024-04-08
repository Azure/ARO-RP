package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (f *frontend) putOrPatchOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var header http.Header
	var b []byte

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := api.GetCorrelationDataFromCtx(r.Context())
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic
	originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
	referer := r.Header.Get("Referer")

	subId := chi.URLParam(r, "subscriptionId")
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")

	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchOpenShiftCluster(ctx, log, body, correlationData, systemData, r.URL.Path, originalPath, r.Method, referer, &header, f.apis[apiVersion].OpenShiftClusterConverter, f.apis[apiVersion].OpenShiftClusterStaticValidator, subId, resourceProviderNamespace, apiVersion)
		return err
	})

	frontendOperationResultLog(log, r.Method, err)
	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchOpenShiftCluster(ctx context.Context, log *logrus.Entry, body []byte, correlationData *api.CorrelationData, systemData *api.SystemData, path, originalPath, method, referer string, header *http.Header, converter api.OpenShiftClusterConverter, staticValidator api.OpenShiftClusterStaticValidator, subId, resourceProviderNamespace string, apiVersion string) ([]byte, error) {
	subscription, err := f.validateSubscriptionState(ctx, path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}
	isCreate := doc == nil

	if isCreate {
		originalR, err := azure.ParseResourceID(originalPath)
		if err != nil {
			return nil, err
		}

		doc = &api.OpenShiftClusterDocument{
			ID:  f.dbOpenShiftClusters.NewUUID(),
			Key: path,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:   originalPath,
				Name: originalR.ResourceName,
				Type: originalR.Provider + "/" + originalR.ResourceType,
				Properties: api.OpenShiftClusterProperties{
					ArchitectureVersion: version.InstallArchitectureVersion,
					ProvisioningState:   api.ProvisioningStateSucceeded,
					CreatedAt:           f.now().UTC(),
					CreatedBy:           version.GitCommit,
					ProvisionedBy:       version.GitCommit,
				},
			},
		}
		if !f.env.IsLocalDevelopmentMode() /* not local dev or CI */ {
			doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled = true
		}
	}

	doc.CorrelationData = correlationData

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return nil, err
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed {
		switch doc.OpenShiftCluster.Properties.FailedProvisioningState {
		case api.ProvisioningStateCreating:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose creation failed. Delete the cluster.")
		case api.ProvisioningStateUpdating:
			// allow: a previous failure to update should not prevent a new
			// operation.
		case api.ProvisioningStateDeleting:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose deletion failed. Delete the cluster.")
		default:
			return nil, fmt.Errorf("unexpected failedProvisioningState %q", doc.OpenShiftCluster.Properties.FailedProvisioningState)
		}
	}

	// If Put or Patch is executed we will enrich document with cluster data.
	if !isCreate {
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		f.clusterEnricher.Enrich(timeoutCtx, log, doc.OpenShiftCluster)
	}

	var ext interface{}
	switch method {
	// In case of PUT we will take customer request payload and store into database
	// Our base structure for unmarshal is skeleton document with values we
	// think is required. We expect payload to have everything else required.
	case http.MethodPut:
		ext = converter.ToExternal(&api.OpenShiftCluster{
			ID:   doc.OpenShiftCluster.ID,
			Name: doc.OpenShiftCluster.Name,
			Type: doc.OpenShiftCluster.Type,
			Properties: api.OpenShiftClusterProperties{
				ProvisioningState: doc.OpenShiftCluster.Properties.ProvisioningState,
				ClusterProfile: api.ClusterProfile{
					PullSecret: doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret,
					Version:    doc.OpenShiftCluster.Properties.ClusterProfile.Version,
				},
				ServicePrincipalProfile: api.ServicePrincipalProfile{
					ClientSecret: doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret,
				},
			},
			SystemData: doc.OpenShiftCluster.SystemData,
		})

	// In case of PATCH we take current cluster document, which is enriched
	// from the cluster and use it as base for unmarshal. So customer can
	// provide single field json to be updated in the database.
	// Patch should be used for updating individual fields of the document.
	case http.MethodPatch:
		ext = converter.ToExternal(doc.OpenShiftCluster)
		converter.ExternalNoReadOnly(ext)
	}

	err = json.Unmarshal(body, &ext)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	if isCreate {
		converter.ToInternal(ext, doc.OpenShiftCluster)
		err = f.ValidateNewCluster(ctx, subscription, doc.OpenShiftCluster, staticValidator, ext, path)
		if err != nil {
			return nil, err
		}
	} else {
		err = staticValidator.Static(ext, doc.OpenShiftCluster, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), path)
		if err != nil {
			return nil, err
		}
	}

	oldID, oldName, oldType, oldSystemData := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type, doc.OpenShiftCluster.SystemData
	converter.ToInternal(ext, doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type, doc.OpenShiftCluster.SystemData = oldID, oldName, oldType, oldSystemData

	// This will update systemData from the values in the header. Old values, which
	// is not provided in the header must be preserved
	f.systemDataClusterDocEnricher(doc, systemData)

	if isCreate {
		err = f.validateInstallVersion(ctx, doc.OpenShiftCluster)
		if err != nil {
			return nil, err
		}

		// on create, make the cluster resourcegroup ID lower case to work
		// around LB/PLS bug
		doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID = strings.ToLower(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)

		doc.ClusterResourceGroupIDKey = strings.ToLower(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
		doc.ClientIDKey = strings.ToLower(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID)
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating

		doc.Bucket, err = f.bucketAllocator.Allocate()
		if err != nil {
			return nil, err
		}
	} else {
		setUpdateProvisioningState(doc, apiVersion)
	}

	// SetDefaults will set defaults on cluster document
	api.SetDefaults(doc, operator.DefaultOperatorFlags)

	doc.AsyncOperationID, err = f.newAsyncOperation(ctx, subId, resourceProviderNamespace, doc)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(referer)
	if err != nil {
		return nil, err
	}

	u.Path = f.operationsPath(subId, resourceProviderNamespace, doc.AsyncOperationID)
	*header = http.Header{
		"Azure-AsyncOperation": []string{u.String()},
	}

	if isCreate {
		newdoc, err := f.dbOpenShiftClusters.Create(ctx, doc)
		if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
			return nil, f.validateOpenShiftUniqueKey(ctx, doc)
		}
		doc = newdoc
	} else {
		doc, err = f.dbOpenShiftClusters.Update(ctx, doc)
	}
	if err != nil {
		return nil, err
	}

	// We remove sensitive data from document to prevent sensitive data being
	// returned to the customer.
	doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	// We don't return enriched worker profile data on PUT/PATCH operations
	doc.OpenShiftCluster.Properties.WorkerProfilesStatus = nil

	b, err := json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
	if err != nil {
		return nil, err
	}

	if isCreate {
		err = statusCodeError(http.StatusCreated)
	}
	return b, err
}

// enrichClusterSystemData will selectively overwrite systemData fields based on
// arm inputs
func enrichClusterSystemData(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
	if systemData == nil {
		return
	}
	if systemData.CreatedAt != nil {
		doc.OpenShiftCluster.SystemData.CreatedAt = systemData.CreatedAt
	}
	if systemData.CreatedBy != "" {
		doc.OpenShiftCluster.SystemData.CreatedBy = systemData.CreatedBy
	}
	if systemData.CreatedByType != "" {
		doc.OpenShiftCluster.SystemData.CreatedByType = systemData.CreatedByType
	}
	if systemData.LastModifiedAt != nil {
		doc.OpenShiftCluster.SystemData.LastModifiedAt = systemData.LastModifiedAt
	}
	if systemData.LastModifiedBy != "" {
		doc.OpenShiftCluster.SystemData.LastModifiedBy = systemData.LastModifiedBy
	}
	if systemData.LastModifiedByType != "" {
		doc.OpenShiftCluster.SystemData.LastModifiedByType = systemData.LastModifiedByType
	}
}

func (f *frontend) ValidateNewCluster(ctx context.Context, subscription *api.SubscriptionDocument, cluster *api.OpenShiftCluster, staticValidator api.OpenShiftClusterStaticValidator, ext interface{}, path string) error {
	err := staticValidator.Static(ext, nil, f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), path)
	if err != nil {
		return err
	}

	err = f.skuValidator.ValidateVMSku(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, cluster)
	if err != nil {
		return err
	}

	err = f.quotaValidator.ValidateQuota(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID, cluster)
	if err != nil {
		return err
	}

	err = f.providersValidator.ValidateProviders(ctx, f.env.Environment(), f.env, subscription.ID, subscription.Subscription.Properties.TenantID)
	if err != nil {
		return err
	}

	return nil
}

// setUpdateProvisioningState Sets either the admin update or update provisioning state
func setUpdateProvisioningState(doc *api.OpenShiftClusterDocument, apiVersion string) {
	switch apiVersion {
	case admin.APIVersion:
		adminUpdateProvisioningState(doc)
	default:
		updateProvisioningState(doc)
	}
}

// Non-admin update (ex: customer cluster update)
func updateProvisioningState(doc *api.OpenShiftClusterDocument) {
	doc.OpenShiftCluster.Properties.LastProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
	doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
	doc.Dequeues = 0
}

// Admin update (ex: cluster maintenance)
func adminUpdateProvisioningState(doc *api.OpenShiftClusterDocument) {
	if doc.OpenShiftCluster.Properties.MaintenanceTask.IsMaintenanceOngoingTask() {
		doc.OpenShiftCluster.Properties.LastProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
		doc.OpenShiftCluster.Properties.LastAdminUpdateError = ""
		doc.Dequeues = 0

		// Set the maintenance to ongoing so we emit the appropriate signal to customerss
		if doc.OpenShiftCluster.Properties.MaintenanceState == api.MaintenanceStatePending {
			doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePlanned
		} else {
			doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateUnplanned
		}
	} else {
		// No default needed since we're using an enum
		switch doc.OpenShiftCluster.Properties.MaintenanceTask {
		case api.MaintenanceTaskPending:
			doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStatePending
		case api.MaintenanceTaskNone:
			doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateNone
		case api.MaintenanceTaskCustomerActionNeeded:
			doc.OpenShiftCluster.Properties.MaintenanceState = api.MaintenanceStateCustomerActionNeeded
		}

		// This enables future admin update actions with body `{}` to succeed
		doc.OpenShiftCluster.Properties.MaintenanceTask = ""
	}
}
