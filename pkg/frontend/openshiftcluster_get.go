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
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
)

func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	b, err := f._getOpenShiftCluster(ctx, r, f.apis[vars["api-version"]].OpenShiftClusterConverter())

	reply(log, w, nil, b, err)
}

func (f *frontend) _getOpenShiftCluster(ctx context.Context, r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	vars := mux.Vars(r)

	doc, err := f.db.OpenShiftClusters.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	f.ocEnricher.Enrich(timeoutCtx, doc.OpenShiftCluster)

	redactedPS, err := pullsecret.Redacted(string(doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret))
	if err != nil {
		doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""
	} else {
		doc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = api.SecureString(redactedPS)
	}
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
}
