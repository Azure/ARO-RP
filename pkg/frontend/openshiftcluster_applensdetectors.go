package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
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
	a, err := f._createAppLensActionsFactory(ctx, r, log)
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
	a, err := f._createAppLensActionsFactory(ctx, r, log)
	if err != nil {
		return nil, err
	}

	detectorId := chi.URLParam(r, "detectorId")
	return a.AppLensGetDetector(ctx, detectorId)
}

func (f *frontend) _createAppLensActionsFactory(ctx context.Context, r *http.Request, log *logrus.Entry) (adminactions.AppLensActions, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	resourceID := strings.TrimSuffix(r.URL.Path, "/detectors")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	a, err := f.appLensActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	return a, nil
}
