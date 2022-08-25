package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) deleteClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	err := f._deleteClusterManagerConfiguration(ctx, log, r)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		err = statusCodeError(http.StatusNoContent)
	case err == nil:
		err = statusCodeError(http.StatusAccepted)
	}
	reply(log, w, nil, nil, err)
}

func (f *frontend) _deleteClusterManagerConfiguration(ctx context.Context, log *logrus.Entry, r *http.Request) error {
	vars := mux.Vars(r)

	doc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	doc.ClusterManagerConfiguration.Deleting = true
	_, err = f.dbClusterManagerConfiguration.Update(ctx, doc)
	if err != nil {
		return err
	}

	err = f.dbClusterManagerConfiguration.Delete(ctx, doc)
	return err
}
