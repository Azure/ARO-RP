package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (a *adminactions) ResourcesList(ctx context.Context) ([]byte, error) {

	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	vNetID, _, err := subnet.Split(a.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	resources, err := a.resourcesClient.List(ctx, fmt.Sprintf("resourceGroup eq '%s'", clusterRGName), "", nil)
	if err != nil {
		return nil, err
	}

	armResources := make([]arm.Resource, 0, len(resources))
	{
		// get customer vnet and append it to the list

		r, err := azure.ParseResourceID(vNetID)
		if err != nil {
			return nil, err
		}

		vNet, err := a.vNetClient.Get(ctx, r.ResourceGroup, r.ResourceName, "")
		if err != nil {
			return nil, err
		}
		armResources = append(armResources, arm.Resource{
			Resource: vNet,
		})
	}

	for _, res := range resources {
		apiVersion, err := azureclient.APIVersionForType(*res.Type)
		if err != nil {
			return nil, err
		}
		switch *res.Type {
		case "Microsoft.Compute/virtualMachines":
			vm, err := a.vmClient.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
			if err != nil {
				return nil, err
			}
			armResources = append(armResources, arm.Resource{
				Resource: vm,
			})
		default:
			gr, err := a.resourcesClient.GetByID(ctx, *res.ID, apiVersion)
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
