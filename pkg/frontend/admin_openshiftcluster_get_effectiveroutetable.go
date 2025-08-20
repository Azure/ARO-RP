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
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenshiftClusterEffectiveRouteTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Use filepath.Dir to get the cluster resource path (same as original)
	r.URL.Path = filepath.Dir(r.URL.Path)

	e, err := f._getOpenshiftClusterEffectiveRouteTable(ctx, r)

	adminReply(log, w, nil, e, err)
}

func (f *frontend) _getOpenshiftClusterEffectiveRouteTable(ctx context.Context, r *http.Request) ([]byte, error) {
	resType := chi.URLParam(r, "resourceType")
	resName := chi.URLParam(r, "resourceName")
	resGroupName := chi.URLParam(r, "resourceGroupName")

	// Extract NIC name from query parameters (this is still required as it's not in the cluster doc)
	nicName := r.URL.Query().Get("nic")
	if nicName == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name is required")
	}

	// Extract resource ID from path
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	if resourceID == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Resource ID is required")
	}

	// Get cluster document from database
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.",
				resType, resName, resGroupName))
	case err != nil:
		return nil, err
	}

	// Get subscription document to obtain tenant ID for authentication
	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to retrieve subscription document: %v", err))
	}

	// Create AzureActions using the existing factory pattern
	azureActions, err := f.azureActionsFactory(nil, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create Azure actions: %v", err))
	}

	// Use AzureActions to get effective route table
	jsonData, err := azureActions.GetEffectiveRouteTable(ctx, nicName)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to retrieve effective route table: %v", err))
	}

	return jsonData, nil
}
