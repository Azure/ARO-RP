package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putOrPatchOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var header http.Header
	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putOrPatchOpenShiftCluster(r, &header, api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClusterToInternal), api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClusterToExternal))
		return err
	})

	reply(log, w, header, b, err)
}

func (f *frontend) _putOrPatchOpenShiftCluster(r *http.Request, header *http.Header, internal api.OpenShiftClusterToInternal, external api.OpenShiftClusterToExternal) ([]byte, error) {
	vars := mux.Vars(r)
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)

	subdoc, err := f.validateSubscriptionState(r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.db.OpenShiftClusters.Get(r.URL.Path)
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
				Properties: api.Properties{
					ProvisioningState: api.ProvisioningStateSucceeded,
					// TODO: ResourceGroup should be exposed in external API
					ResourceGroup: vars["resourceName"],
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						TenantID: subdoc.Subscription.Properties.TenantID,
					},
				},
			},
		}
	}

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return nil, err
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed {
		switch doc.OpenShiftCluster.Properties.FailedProvisioningState {
		case api.ProvisioningStateCreating:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose creation failed. Delete the cluster.")
		case api.ProvisioningStateUpdating:
			doc.OpenShiftCluster.Properties.FailedProvisioningState = "" // allow
		case api.ProvisioningStateDeleting:
			return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose deletion failed. Delete the cluster.")
		default:
			return nil, fmt.Errorf("unexpected failedProvisioningState %q", doc.OpenShiftCluster.Properties.FailedProvisioningState)
		}
	}

	var ext interface{}
	switch r.Method {
	case http.MethodPut:
		ext = external.OpenShiftClusterToExternal(&api.OpenShiftCluster{
			ID:   doc.OpenShiftCluster.ID,
			Name: doc.OpenShiftCluster.Name,
			Type: doc.OpenShiftCluster.Type,
			Properties: api.Properties{
				ProvisioningState: doc.OpenShiftCluster.Properties.ProvisioningState,
			},
		})

	case http.MethodPatch:
		ext = external.OpenShiftClusterToExternal(doc.OpenShiftCluster)
	}

	err = json.Unmarshal(body, &ext)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	if isCreate {
		err = internal.ValidateOpenShiftCluster(f.env.Location(), r.URL.Path, ext, nil)
	} else {
		err = internal.ValidateOpenShiftCluster(f.env.Location(), r.URL.Path, ext, doc.OpenShiftCluster)
	}
	if err != nil {
		return nil, err
	}

	oldID, oldName, oldType := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type
	internal.OpenShiftClusterToInternal(ext, doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type = oldID, oldName, oldType

	if isCreate {
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating
	} else {
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
		doc.Dequeues = 0
	}

	err = internal.ValidateOpenShiftClusterDynamic(r.Context(), f.env.FPAuthorizer, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	doc.AsyncOperationID, err = f.newAsyncOperation(r, doc)
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
		doc, err = f.db.OpenShiftClusters.Create(doc)
	} else {
		doc, err = f.db.OpenShiftClusters.Update(doc)
	}
	if err != nil {
		return nil, err
	}

	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	b, err := json.MarshalIndent(external.OpenShiftClusterToExternal(doc.OpenShiftCluster), "", "    ")
	if err != nil {
		return nil, err
	}

	if isCreate {
		err = statusCodeError(http.StatusCreated)
	}
	return b, err
}
