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
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (f *frontend) putOrPatchOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	_, developmentMode := f.env.(env.Dev)

	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var header http.Header
	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchOpenShiftCluster(ctx, r, &header, f.apis[vars["api-version"]].OpenShiftClusterConverter(), f.apis[vars["api-version"]].OpenShiftClusterStaticValidator(f.env.Location(), f.env.Domain(), developmentMode, r.URL.Path))
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchOpenShiftCluster(ctx context.Context, r *http.Request, header *http.Header, converter api.OpenShiftClusterConverter, staticValidator api.OpenShiftClusterStaticValidator) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	correlationData := r.Context().Value(middleware.ContextKeyCorrelationData).(*api.CorrelationData)

	subdoc, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.db.OpenShiftClusters.Get(ctx, r.URL.Path)
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
			ID:  uuid.NewV4().String(),
			Key: r.URL.Path,
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:   originalPath,
				Name: originalR.ResourceName,
				Type: originalR.Provider + "/" + originalR.ResourceType,
				Properties: api.OpenShiftClusterProperties{
					ProvisioningState: api.ProvisioningStateSucceeded,
					ClusterProfile: api.ClusterProfile{
						Version: version.OpenShiftVersion,
					},
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						TenantID: subdoc.Subscription.Properties.TenantID,
					},
				},
			},
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

	if !isCreate {
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		f.ocEnricher.Enrich(timeoutCtx, doc.OpenShiftCluster)
	}

	var ext interface{}
	switch r.Method {
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

	oldID, oldName, oldType := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type
	converter.ToInternal(ext, doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type = oldID, oldName, oldType

	if isCreate {
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
		} else {
			doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
		}
		doc.Dequeues = 0
	}

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
		newdoc, err := f.db.OpenShiftClusters.Create(ctx, doc)
		if cosmosdb.IsErrorStatusCode(err, http.StatusPreconditionFailed) {
			return nil, f.validateOpenShiftUniqueKey(ctx, doc)
		}
		doc = newdoc
	} else {
		doc, err = f.db.OpenShiftClusters.Update(ctx, doc)
	}
	if err != nil {
		return nil, err
	}

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
