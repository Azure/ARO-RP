package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	utilarmnetwork "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
)

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

	// Create Azure credentials using the environment's helper
	credential, err := f.env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID, nil)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create Azure credentials: %v", err))
	}

	// Create ARM network interfaces client using ARO-RP utility
	interfacesClient, err := utilarmnetwork.NewInterfacesClient(subscriptionDoc.ID, credential, nil)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create network interfaces client: %v", err))
	}

	// Use the cluster document's ClusterResourceGroup instead of the cluster resource group
	clusterResourceGroup := doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
	if clusterResourceGroup == "" {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			"Cluster resource group not found in cluster document")
	}

	// Extract resource group name from resource group ID
	// ClusterResourceGroup format: /subscriptions/{sub}/resourceGroups/{rg}
	parts := strings.Split(clusterResourceGroup, "/")
	if len(parts) < 5 || parts[3] != "resourceGroups" {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			"Invalid cluster resource group format")
	}
	clusterResourceGroupName := parts[4]

	// Call GetEffectiveRouteTable using the cluster resource group
	result, err := interfacesClient.GetEffectiveRouteTableAndWait(ctx, clusterResourceGroupName, nicName, nil)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to retrieve effective route table: %v", err))
	}

	// Marshal the result to JSON
	jsonData, err := result.MarshalJSON()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to serialize route table data: %v", err))
	}

	return jsonData, nil
}
