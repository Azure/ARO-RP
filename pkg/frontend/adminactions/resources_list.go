package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (a *azureActions) ResourcesList(ctx context.Context) ([]byte, error) {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	resources, err := a.resources.ListByResourceGroup(ctx, clusterRGName, "", "", nil)
	if err != nil {
		return nil, err
	}

	// +4 because we expect +2 subnet and +1 vnets and optional +1 diskEncryptionSet
	armResources := make([]arm.Resource, 0, len(resources)+4)
	armResources, err = a.appendAzureNetworkResources(ctx, armResources)
	if err != nil {
		a.log.Warnf("error when getting network resources: %s", err)
	}
	armResources, err = a.appendAzureDiskEncryptionSetResources(ctx, armResources)
	if err != nil {
		a.log.Warnf("error when getting DiskEncryptionSet resources: %s", err)
	}

	for _, res := range resources {
		apiVersion := azureclient.APIVersion(*res.Type)
		if apiVersion == "" {
			// If custom resource types, or any we don't have listed in pkg/util/azureclient/apiversions.go,
			// are returned, then skip over them instead of returning an error, otherwise it results in an
			// HTTP 500 and prevents the known resource types from being returned.
			a.log.Warnf("API version not found for type %q", *res.Type)
			continue
		}
		switch *res.Type {
		case "Microsoft.Compute/virtualMachines":
			vm, err := a.virtualMachines.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
			if err != nil {
				a.log.Warn(err) // can happen when the ARM cache is lagging
				armResources = append(armResources, arm.Resource{
					Resource: res,
				})
				continue
			}
			armResources = append(armResources, arm.Resource{
				Resource: vm,
			})
		default:
			gr, err := a.resources.GetByID(ctx, *res.ID, apiVersion)
			if err != nil {
				a.log.Warn(err) // can happen when the ARM cache is lagging
				armResources = append(armResources, arm.Resource{
					Resource: res,
				})
				continue
			}
			armResources = append(armResources, arm.Resource{
				Resource: gr,
			})
		}
	}

	return json.Marshal(armResources)
}

func (a *azureActions) appendAzureNetworkResources(ctx context.Context, armResources []arm.Resource) ([]arm.Resource, error) {
	vNetID, _, err := subnet.Split(a.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vNetID)
	if err != nil {
		return armResources, err
	}

	vnet, err := a.virtualNetworks.Get(ctx, r.ResourceGroup, r.ResourceName, "")
	if err != nil {
		return armResources, err
	}
	armResources = append(armResources, arm.Resource{
		Resource: vnet,
	})
	if vnet.Subnets != nil {
		for _, snet := range *vnet.Subnets {
			//we already have the VNet resource, filtering subnets instead of fetching them individually with a SubnetClient
			interestingSubnet := (*snet.ID == a.oc.Properties.MasterProfile.SubnetID)
			for _, wProfile := range a.oc.Properties.WorkerProfiles {
				interestingSubnet = interestingSubnet || (*snet.ID == wProfile.SubnetID)
			}
			if !interestingSubnet {
				continue
			}
			//by this time the snet subnet is used in a Master or Worker profile
			if snet.RouteTable != nil {
				r, err := azure.ParseResourceID(*snet.RouteTable.ID)
				if err != nil {
					a.log.Warnf("skipping route table '%s' due to ID parse error: %s", *snet.RouteTable.ID, err)
					continue
				}
				rt, err := a.routeTables.Get(ctx, r.ResourceGroup, r.ResourceName, "")
				if err != nil {
					a.log.Warnf("skipping route table '%s' due to Get error: %s", *snet.RouteTable.ID, err)
					continue
				}
				armResources = append(armResources, arm.Resource{
					Resource: rt,
				})
			}
		}
	}

	return armResources, nil
}

func (a *azureActions) appendAzureDiskEncryptionSetResources(ctx context.Context, armResources []arm.Resource) ([]arm.Resource, error) {
	// possible for there to be no DiskEncryptionSet, if so, ignore
	if a.oc.Properties.MasterProfile.DiskEncryptionSetID == "" {
		return armResources, nil
	}

	r, err := azure.ParseResourceID(a.oc.Properties.MasterProfile.DiskEncryptionSetID)
	if err != nil {
		return armResources, err
	}

	diskEncryptionSets, err := a.diskEncryptionSets.Get(ctx, r.ResourceGroup, r.ResourceName)
	if err != nil {
		return armResources, err
	}

	armResources = append(armResources, arm.Resource{
		Resource: diskEncryptionSets,
	})

	return armResources, nil
}
