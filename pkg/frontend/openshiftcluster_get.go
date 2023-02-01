package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]

	doc, err := f.dbOpenShiftClusters.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ocEnricher := f.ocEnricherFactory(log, f.env, f.m)
	ocEnricher.Enrich(timeoutCtx, doc.OpenShiftCluster)

	doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
}
