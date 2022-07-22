package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	b, err := f._getClusterManagerConfiguration(ctx, log, r, f.apis[vars["api-version"]].ClusterManagerConfigurationConverter())
	reply(log, w, nil, b, err)
}

func (f *frontend) _getClusterManagerConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.ClusterManagerConfigurationConverter) ([]byte, error) {
	vars := mux.Vars(r)

	doc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' was not found.", vars["resourceType"], vars["resourceName"], vars["clusterManagerKind"], vars["ocmResourceName"])
	case err != nil:
		return nil, err
	}

	ext, err := converter.ToExternal(doc.ClusterManagerConfiguration)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(ext, "", "    ")
}
