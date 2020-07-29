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

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
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

	b, err := f._listAdminOpenShiftClusterResources(ctx, r)
	if err == nil {
		b, err = adminJmespathFilter(b, jpath)
	}

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

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := f.env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	resourcesClient := f.resourcesClientFactory(subscriptionDoc.ID, fpAuthorizer)
	vmClient := f.vmClientFactory(subscriptionDoc.ID, fpAuthorizer)
	vnetClient := f.vNetClientFactory(subscriptionDoc.ID, fpAuthorizer)

	clusterResourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	resources, err := resourcesClient.List(ctx, fmt.Sprintf("resourceGroup eq '%s'", clusterResourceGroup), "", nil)
	if err != nil {
		return nil, err
	}

	armResources := make([]arm.Resource, 0, len(resources))
	{ // get customer vnet and append it to the list
		vnetID, _, err := subnet.Split(doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
		if err != nil {
			return nil, err
		}

		r, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return nil, err
		}

		vnet, err := vnetClient.Get(ctx, r.ResourceGroup, r.ResourceName, "")
		if err != nil {
			return nil, err
		}
		armResources = append(armResources, arm.Resource{
			Resource: vnet,
		})
	}

	for _, res := range resources {
		apiVersion, err := azureclient.APIVersionForType(*res.Type)
		if err != nil {
			return nil, err
		}
		switch *res.Type {
		case "Microsoft.Compute/virtualMachines":
			vm, err := vmClient.Get(ctx, clusterResourceGroup, *res.Name, mgmtcompute.InstanceView)
			if err != nil {
				return nil, err
			}
			armResources = append(armResources, arm.Resource{
				Resource: vm,
			})
		default:
			gr, err := resourcesClient.GetByID(ctx, *res.ID, apiVersion)
			if err != nil {
				return nil, err
			}
			armResources = append(armResources, arm.Resource{
				Resource: gr,
			})
		}
	}

	return json.Marshal(armResources)
}
