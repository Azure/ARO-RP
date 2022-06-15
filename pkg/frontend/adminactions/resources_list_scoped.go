package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (a *azureActions) ResourcesListScoped(ctx context.Context, resourceType string) ([]byte, error) {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	filter := fmt.Sprintf("resourceType eq '%s'", resourceType)
	resources, err := a.resources.ListByResourceGroup(ctx, clusterRGName, filter, "", nil)
	if err != nil {
		return nil, err
	}

	armResources := make([]arm.Resource, 0)
	for _, res := range resources {
		if *res.Type == "Microsoft.Compute/virtualMachines" {
			vm, err := a.virtualMachines.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
			if err != nil {
				a.log.Warn(err)
				armResources = append(armResources, arm.Resource{
					Resource: res,
				})
				continue
			}
			armResources = append(armResources, arm.Resource{
				Resource: vm,
			})
		}

		armResources = append(armResources, arm.Resource{
			Resource: res,
		})
	}

	return json.Marshal(armResources)
}
