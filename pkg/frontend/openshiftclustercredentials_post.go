package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postOpenShiftClusterCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceType := chi.URLParam(r, "resourceType")
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")

	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	if f.apis[apiVersion].OpenShiftClusterCredentialsConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", resourceType, resourceProviderNamespace, apiVersion)
		return
	}

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) > 0 && !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._postOpenShiftClusterCredentials(ctx, r, f.apis[apiVersion].OpenShiftClusterCredentialsConverter)

	reply(log, w, nil, b, err)
}

func (f *frontend) _postOpenShiftClusterCredentials(ctx context.Context, r *http.Request, converter api.OpenShiftClusterCredentialsConverter) ([]byte, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered)
	if err != nil {
		return nil, err
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return nil, err
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateCreating ||
		doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateDeleting ||
		doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed && doc.OpenShiftCluster.Properties.FailedProvisioningState == api.ProvisioningStateCreating ||
		doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed && doc.OpenShiftCluster.Properties.FailedProvisioningState == api.ProvisioningStateDeleting {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed in provisioningState '%s'.", doc.OpenShiftCluster.Properties.ProvisioningState)
	}

	doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
}
