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
	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (f *frontend) putOrPatchOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var header http.Header
	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchOpenShiftCluster(ctx, log, r, &header, f.apis[vars["api-version"]].OpenShiftClusterConverter(), f.apis[vars["api-version"]].OpenShiftClusterStaticValidator(f.env.Location(), f.env.Domain(), f.env.FeatureIsSet(env.FeatureRequireD2sV3Workers), r.URL.Path))
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchOpenShiftCluster(ctx context.Context, log *logrus.Entry, r *http.Request, header *http.Header, converter api.OpenShiftClusterConverter, staticValidator api.OpenShiftClusterStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)
	systemData, _ := r.Context().Value(middleware.ContextKeySystemData).(*api.SystemData) // don't panic

	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, r.URL.Path)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	isCreate := doc == nil

	if isCreate {
		originalPath := r.Context().Value(middleware.ContextKeyOriginalPath).(string)
		originalR, err := azure.ParseResourceID(originalPath)
		if err != nil {
			return nil, err
		}

		doc = &api.OpenShiftClusterDocument{
			ID:  uuid.Must(uuid.NewV4()).String(),
			Key: r.URL.Path,
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
					ClusterProfile: api.ClusterProfile{
						Version: version.InstallStream.Version.String(),
					},
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

		ocEnricher := f.ocEnricherFactory(log, f.env, f.m)
		ocEnricher.Enrich(timeoutCtx, doc.OpenShiftCluster)
	}

	var ext interface{}
	switch r.Method {
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
		})

		// In case of PATCH we take current cluster document, which is enriched
		// from the cluster and use it as base for unmarshal. So customer can
		// provide single field json to be updated in the database.
		// Patch should be used for updating individual fields of the document.
	case http.MethodPatch:
		ext = converter.ToExternal(doc.OpenShiftCluster)
	}

	err = json.Unmarshal(body, &ext)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	if isCreate {
		err = staticValidator.Static(ext, nil)
	} else {
		err = staticValidator.Static(ext, doc.OpenShiftCluster)
	}
	if err != nil {
		return nil, err
	}

	oldID, oldName, oldType, oldSystemData := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type, doc.OpenShiftCluster.SystemData
	converter.ToInternal(ext, doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type, doc.OpenShiftCluster.SystemData = oldID, oldName, oldType, oldSystemData

	// This will update systemData from the values in the header. Old values, which
	// is not provided in the header must be preserved
	f.systemDataEnricher(doc, systemData)

	if isCreate {
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
		doc.OpenShiftCluster.Properties.LastProvisioningState = doc.OpenShiftCluster.Properties.ProvisioningState

		// TODO: Get rid of the special case
		vars := mux.Vars(r)
		if vars["api-version"] == admin.APIVersion {
			doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateAdminUpdating
			doc.OpenShiftCluster.Properties.LastAdminUpdateError = ""
		} else {
			doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
		}
		doc.Dequeues = 0
	}

	// SetDefaults will set defaults on cluster document
	api.SetDefaults(doc)

	doc.AsyncOperationID, err = f.newAsyncOperation(ctx, r, doc)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(r.Header.Get("Referer"))
	if err != nil {
		return nil, err
	}

	u.Path = f.operationsPath(r, doc.AsyncOperationID)
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

	b, err := json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
	if err != nil {
		return nil, err
	}

	if isCreate {
		err = statusCodeError(http.StatusCreated)
	}
	return b, err
}

// enrichSystemData will selectively overwrite systemData fields based on
// arm inputs
func enrichSystemData(doc *api.OpenShiftClusterDocument, systemData *api.SystemData) {
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
