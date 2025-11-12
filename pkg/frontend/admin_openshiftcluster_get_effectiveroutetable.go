package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenShiftClusterEffectiveRouteTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Use filepath.Dir to get the cluster resource path (same as original)
	r.URL.Path = filepath.Dir(r.URL.Path)

	e, err := f._getOpenshiftClusterEffectiveRouteTable(ctx, r)

	adminReply(log, w, nil, e, err)
}

func (f *frontend) _getOpenshiftClusterEffectiveRouteTable(ctx context.Context, r *http.Request) ([]byte, error) {
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resType := chi.URLParam(r, "resourceType")
	resName := chi.URLParam(r, "resourceName")
	resGroupName := chi.URLParam(r, "resourceGroupName")

	// Extract all query parameters
	queryParams := r.URL.Query()
	nicName := strings.TrimSpace(queryParams.Get("nic"))
	subId := strings.TrimSpace(queryParams.Get("subid"))
	rgn := strings.TrimSpace(queryParams.Get("rgn"))

	// Validate required NIC parameter
	if nicName == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name is required")
	}
	
	// Basic NIC name validation (Azure naming rules)
	if len(nicName) > 80 || len(nicName) < 1 {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "nic", "Network interface name must be between 1-80 characters")
	}

	// Validate subid parameter matches URL path subscription ID (if provided)
	if subId != "" && subId != chi.URLParam(r, "subscriptionId") {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "subid", 
			"Query parameter 'subid' does not match URL path subscription ID")
	}

	// Validate rgn parameter matches URL path resource group name (if provided)
	if rgn != "" && rgn != chi.URLParam(r, "resourceGroupName") {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "rgn", 
			"Query parameter 'rgn' does not match URL path resource group name")
	}

	// Extract resource ID from path
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	if resourceID == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Resource ID is required")
	}

	// Validate required URL parameters
	if resType == "" || resName == "" || resGroupName == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "urlParams", 
			"Missing required URL parameters: resourceType, resourceName, or resourceGroupName")
	}

	// Validate subscription ID
	subscriptionId := chi.URLParam(r, "subscriptionId")
	if subscriptionId == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "subscriptionId", "Subscription ID is required")
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
	azureActions, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create Azure actions: %v", err))
	}

	// Create context with timeout for Azure operation (5 minutes)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Log all received inputs for debugging
	log.Infof("Effective route table request - Subscription: %s, ResourceGroup: %s, ResourceType: %s, ResourceName: %s", 
		chi.URLParam(r, "subscriptionId"), resGroupName, resType, resName)
	log.Infof("Query parameters - NIC: %s, subid: %s, rgn: %s", nicName, subId, rgn)
	log.Infof("Processed resourceID: %s", resourceID)

	// Use AzureActions to get effective route table
	jsonData, err := azureActions.GetEffectiveRouteTable(timeoutCtx, nicName)
	if err != nil {
		// Check if it was a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Errorf("Timeout retrieving effective route table for NIC: %s", nicName)
			return nil, api.NewCloudError(http.StatusRequestTimeout, api.CloudErrorCodeInternalServerError, "",
				"Operation timed out while retrieving effective route table")
		}
		log.Errorf("Failed to retrieve effective route table for NIC %s: %v", nicName, err)
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to retrieve effective route table: %v", err))
	}

	// Log success with data size for monitoring
	log.Infof("Successfully retrieved effective route table for NIC: %s, response size: %d bytes", nicName, len(jsonData))

	return jsonData, nil
}
