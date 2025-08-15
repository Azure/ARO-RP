package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	utilarmnetwork "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
)

func (f *frontend) _getOpenshiftClusterEffectiveRouteTable(ctx context.Context, r *http.Request) ([]byte, error) {
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

	// Parse the resource ID to extract subscription and resource group
	resource, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "resourceId", "Invalid resource ID format")
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
				resource.ResourceType, resource.ResourceName, resource.ResourceGroup))
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
	interfacesClient, err := utilarmnetwork.NewInterfacesClient(resource.SubscriptionID, credential, nil)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "",
			fmt.Sprintf("Failed to create network interfaces client: %v", err))
	}

	// Call GetEffectiveRouteTable using the ARO-RP utility pattern
	result, err := interfacesClient.GetEffectiveRouteTableAndWait(ctx, resource.ResourceGroup, nicName, nil)
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
