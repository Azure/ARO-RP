package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAsyncOperationResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	header := http.Header{}
	b, err := f._getAsyncOperationResult(ctx, r, header, f.apis[r.URL.Query().Get(api.APIVersionKey)].OpenShiftClusterConverter)

	reply(log, w, header, b, err)
}

func (f *frontend) _getAsyncOperationResult(ctx context.Context, r *http.Request, header http.Header, converter api.OpenShiftClusterConverter) ([]byte, error) {
	operationId := chi.URLParam(r, "operationId")
	subscriptionId := chi.URLParam(r, "subscriptionId")

	dbAsyncOperations, err := f.dbGroup.AsyncOperations()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	asyncdoc, err := dbAsyncOperations.Get(ctx, operationId)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The entity was not found.")
	case err != nil:
		return nil, err
	}

	resource, err := azure.ParseResourceID(asyncdoc.OpenShiftClusterKey)
	switch {
	case err != nil:
		return nil, err
	case resource.SubscriptionID != subscriptionId:
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The entity was not found.")
	}

	doc, err := dbOpenShiftClusters.Get(ctx, asyncdoc.OpenShiftClusterKey)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	// don't give away the final operation status until it's committed to the
	// database
	if doc != nil && doc.AsyncOperationID == operationId {
		header["Location"] = r.Header["Referer"]
		return nil, statusCodeError(http.StatusAccepted)
	}

	if asyncdoc.OpenShiftCluster == nil {
		return nil, statusCodeError(http.StatusNoContent)
	}

	asyncdoc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""

	if asyncdoc.OpenShiftCluster.Properties.ServicePrincipalProfile != nil {
		asyncdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""
	}
	asyncdoc.OpenShiftCluster.Properties.ClusterProfile.BoundServiceAccountSigningKey = nil

	return json.MarshalIndent(converter.ToExternal(asyncdoc.OpenShiftCluster), "", "    ")
}
