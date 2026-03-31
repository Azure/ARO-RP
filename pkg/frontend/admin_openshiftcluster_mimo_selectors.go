package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/mimo/scheduler"
)

// /admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/selectors
func (f *frontend) getAdminOpenShiftClusterSelectors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	b, err := f._getAdminOpenShiftClusterSelectors(ctx, r, log)
	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClusterSelectors(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	rID, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	dbSubscriptions, err := f.dbGroup.Subscriptions()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf(
				"The Resource '%s/%s' under resource group '%s' was not found.",
				resType, resName, resGroupName))
	case err != nil:
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	subDoc, err := dbSubscriptions.Get(ctx, rID.SubscriptionID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf(
				"The subscription ID '%s' was not found.",
				rID.SubscriptionID))
	case err != nil:
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	selectorData, err := scheduler.ToSelectorData(doc, string(subDoc.Subscription.State))
	if err != nil {
		return nil, err
	}

	return json.Marshal(selectorData)
}
