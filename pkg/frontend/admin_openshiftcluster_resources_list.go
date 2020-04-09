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

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (f *frontend) listAdminOpenShiftClusterResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._listAdminOpenShiftClusterResources(ctx, r)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _listAdminOpenShiftClusterResources(ctx context.Context, r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	resource, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := f.env.FPAuthorizer(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	resourcesClient := f.resourcesClientFactory(resource.SubscriptionID, fpAuthorizer)

	clusterResourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resources, err := resourcesClient.List(ctx, fmt.Sprintf("resourceGroup eq '%s'", clusterResourceGroup), "", nil)
	if err != nil {
		return nil, err
	}

	for i, res := range resources {
		apiVersion, err := azureclient.APIVersionForType(*res.Type)
		if err != nil {
			return nil, err
		}

		gr, err := resourcesClient.GetByID(ctx, *res.ID, apiVersion)
		if err != nil {
			return nil, err
		}

		res.ID = gr.ID
		res.Name = gr.Name
		res.Type = gr.Type
		res.Location = gr.Location
		res.Kind = gr.Kind
		res.Identity = gr.Identity
		res.Properties = gr.Properties
		res.Plan = gr.Plan
		res.Tags = gr.Tags
		res.Sku = gr.Sku
		res.ManagedBy = gr.ManagedBy

		resources[i] = res
	}

	return json.Marshal(resources)
}
