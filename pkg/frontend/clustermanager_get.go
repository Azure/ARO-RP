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

	if disableOCMAPI {
		reply(log, w, nil, []byte("forbidden."), api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "forbidden."))
		return
	}

	b, err := f._getClusterManagerConfiguration(ctx, log, r, f.apis[vars["api-version"]].ClusterManagerConfigurationConverter)
	reply(log, w, nil, b, err)
}

func (f *frontend) _getClusterManagerConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request, converter api.ClusterManagerConfigurationConverter) ([]byte, error) {
	vars := mux.Vars(r)

	doc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	if doc.Deleting {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on a resource marked for deletion.")
	}

	ext, err := converter.ToExternal(doc.ClusterManagerConfiguration)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(ext, "", "    ")
}
