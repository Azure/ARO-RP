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
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) listAdminOpenShiftClusterResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	jpath, err := validateAdminJmespathFilter(r.URL.Query().Get("filter"))
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	b, err := f._listAdminOpenShiftClusterResources(ctx, r, log)
	if err == nil {
		b, err = adminJmespathFilter(b, jpath, "")
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _listAdminOpenShiftClusterResources(
	ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	vars := mux.Vars(r)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	a, err := f.adminActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	return a.ResourcesList(ctx)
}

func (f *frontend) deleteAdminOpenShiftClusterResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._deletedminOpenShiftClusterResources(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _deletedminOpenShiftClusterResources(
	ctx context.Context, r *http.Request, log *logrus.Entry) error {
	vars := mux.Vars(r)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return err
	}

	deleteResourceType, deleteResourceName := r.URL.Query().Get("resourceType"), r.URL.Query().Get("resourceName")

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return err
	}

	a, err := f.adminActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return err
	}

	return a.ResourcesDelete(ctx, deleteResourceName, deleteResourceType)
}
