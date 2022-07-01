package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) listAppLensDetectors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._listAppLensDetectors(ctx, r, log)

	reply(log, w, nil, b, err)
}

func (f *frontend) _listAppLensDetectors(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	a, err := f._createAzureActionsFactory(ctx, r, log)
	if err != nil {
		return nil, err
	}

	return a.AppLensListDetectors(ctx)
}

func (f *frontend) getAppLensDetector(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._appLensDetectors(ctx, r, log)

	reply(log, w, nil, b, err)
}

func (f *frontend) _appLensDetectors(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	a, err := f._createAzureActionsFactory(ctx, r, log)
	if err != nil {
		return nil, err
	}

	vars := mux.Vars(r)
	return a.AppLensGetDetector(ctx, vars["detectorId"])
}

func (f *frontend) _createAzureActionsFactory(ctx context.Context, r *http.Request, log *logrus.Entry) (adminactions.AzureActions, error) {
	vars := mux.Vars(r)

	resourceID := strings.TrimSuffix(r.URL.Path, "/detectors")
	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	return a, nil
}
