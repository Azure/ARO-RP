package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	b, err := f._getOpenShiftCluster(ctx, log, r, f.apis[r.URL.Query().Get(api.APIVersionKey)].OpenShiftClusterConverter)

	frontendOperationResultLog(log, r.Method, err)
	reply(log, w, nil, b, err)
}

func (f *frontend) _getOpenShiftCluster(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	f.clusterEnricher.Enrich(timeoutCtx, log, doc.OpenShiftCluster)

	doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""

	if doc.OpenShiftCluster.Properties.ServicePrincipalProfile != nil {
		doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""
	}
	doc.OpenShiftCluster.Properties.ClusterProfile.BoundServiceAccountSigningKey = nil

	return json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
}
