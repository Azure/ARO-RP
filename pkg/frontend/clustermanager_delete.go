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
	vars := mux.Vars(r)
	var err error

	apiVersion, ocmResourceType := r.URL.Query().Get(api.APIVersionKey), vars["ocmResourceType"]

	err = f.validateOcmResourceType(apiVersion, ocmResourceType)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", err.Error())
		return
	}

	err = f._deleteClusterManagerConfigurationDocument(ctx, log, r)

	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		err = statusCodeError(http.StatusNoContent)
	case err == nil:
		err = statusCodeError(http.StatusOK)
	}

	reply(log, w, nil, nil, err)
}

func (f *frontend) _deleteClusterManagerConfigurationDocument(ctx context.Context, log *logrus.Entry, r *http.Request) error {
	vars := mux.Vars(r)

	_, err := f.validateSubscriptionState(ctx, r.URL.Path, api.SubscriptionStateRegistered, api.SubscriptionStateSuspended, api.SubscriptionStateWarned)
	if err != nil {
		return err
	}

	doc, err := f.dbClusterManagerConfiguration.Get(ctx, r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["ocmResourceType"], vars["ocmResourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	// Right now we are going to assume that the backend will delete the document, we will just mark for deletion.
	doc.Deleting = true
	err = cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		_, err = f.dbClusterManagerConfiguration.Update(ctx, doc)
		return err
	})
	return err
}
