package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

var commaAsByteArray = []byte(",")

func (a *azureActions) GroupResourceList(ctx context.Context) ([]mgmtfeatures.GenericResourceExpanded, error) {
	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	return a.resources.ListByResourceGroup(ctx, clusterRGName, "", "", nil)
}

func (a *azureActions) ResourcesList(ctx context.Context, resources []mgmtfeatures.GenericResourceExpanded, writer io.WriteCloser) error {
	defer writer.Close()

	clusterRGName := stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/')
	writer.Write([]byte("["))
	// +4 because we expect +2 subnet and +1 vnets and optional +1 diskEncryptionSet
	armResources := make([]arm.Resource, 0, 4)
	armResources, err := a.appendAzureNetworkResources(ctx, armResources)
	if err != nil {
		a.log.Warnf("error when getting network resources: %s", err)
	}
	armResources, err = a.appendAzureDiskEncryptionSetResources(ctx, armResources)
	if err != nil {
		a.log.Warnf("error when getting DiskEncryptionSet resources: %s", err)
	}
	hasWritten := false

	for _, resource := range armResources {
		if hasWritten {
			writer.Write(commaAsByteArray)
		}
		a.writeObject(writer, resource)
		hasWritten = true
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

		if hasWritten {
			writer.Write(commaAsByteArray)
		}
		hasWritten = true

		switch *res.Type {
		case "Microsoft.Compute/virtualMachines":
			vm, err := a.virtualMachines.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
			if err != nil {
				a.log.Warn(err) // can happen when the ARM cache is lagging
				a.writeObject(writer, arm.Resource{
					Resource: res,
				})
				continue
			}
			a.writeObject(writer, arm.Resource{
				Resource: vm,
			})
		default:
			gr, err := a.resources.GetByID(ctx, *res.ID, apiVersion)
			if err != nil {
				a.log.Warn(err) // can happen when the ARM cache is lagging
				a.writeObject(writer, arm.Resource{
					Resource: res,
				})
				continue
			}
			a.writeObject(writer, arm.Resource{
				Resource: gr,
			})
		}
	}

	_, err = writer.Write([]byte("]"))

	return err
}

func (a *azureActions) writeObject(writer io.Writer, resource arm.Resource) {
	bytes, err := resource.MarshalJSON()
	if err != nil {
		a.log.Warn(err) // very unlikely , only a handful of cases trigger an error
		// here. and since we get the object from a database , it probably will never happen
		return
	}

	_, err = writer.Write(bytes)
	if err != nil {
		a.log.Warn(err) // can happen if the the connection is closed for example
	}
}

func (a *azureActions) appendAzureNetworkResources(ctx context.Context, armResources []arm.Resource) ([]arm.Resource, error) {
	vNetID, _, err := apisubnet.Split(a.oc.Properties.MasterProfile.SubnetID)
	if err != nil {
		return nil, err
	}

	r, err := azure.ParseResourceID(vNetID)
	if err != nil {
		return armResources, err
	}

	vnet, err := a.virtualNetworks.Get(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return armResources, err
	}
	armResources = append(armResources, arm.Resource{
		Resource: vnet.VirtualNetwork,
	})
	if vnet.Properties.Subnets == nil {
		return armResources, nil
	}
	for _, snet := range vnet.Properties.Subnets {
		// we already have the VNet resource, filtering subnets instead of fetching them individually with a SubnetClient
		interestingSubnet := (*snet.ID == a.oc.Properties.MasterProfile.SubnetID)
		workerProfiles, _ := api.GetEnrichedWorkerProfiles(a.oc.Properties)

		for _, wProfile := range workerProfiles {
			interestingSubnet = interestingSubnet || (*snet.ID == wProfile.SubnetID)
		}
		if !interestingSubnet {
			continue
		}
		// by this time the snet subnet is used in a Master or Worker profile
		if snet.Properties.RouteTable != nil {
			r, err := azure.ParseResourceID(*snet.Properties.RouteTable.ID)
			if err != nil {
				a.log.Warnf("skipping route table '%s' due to ID parse error: %s", *snet.Properties.RouteTable.ID, err)
				continue
			}
			rt, err := a.routeTables.Get(ctx, r.ResourceGroup, r.ResourceName, nil)
			if err != nil {
				a.log.Warnf("skipping route table '%s' due to Get error: %s", *snet.Properties.RouteTable.ID, err)
				continue
			}
			armResources = append(armResources, arm.Resource{
				Resource: rt.RouteTable,
			})
		}
		// Due to BYO NSGs not belonging to the managed resource group, we would like to list them as well
		if snet.Properties.NetworkSecurityGroup == nil {
			continue
		}
		nsgID, err := azure.ParseResourceID(*snet.Properties.NetworkSecurityGroup.ID)
		if err != nil {
			a.log.Warnf("skipping NSG '%s' due to ID parse error: %s", *snet.Properties.NetworkSecurityGroup.ID, err)
			continue
		}
		if nsgID.ResourceGroup == stringutils.LastTokenByte(a.oc.Properties.ClusterProfile.ResourceGroupID, '/') {
			continue
		}
		nsg, err := a.securityGroups.Get(ctx, nsgID.ResourceGroup, nsgID.ResourceName, nil)
		if err != nil {
			a.log.Warnf("skipping NSG '%s' due to Get error: %s", *snet.Properties.NetworkSecurityGroup.ID, err)
			continue
		}
		armResources = append(armResources, arm.Resource{
			Resource: nsg.SecurityGroup,
		})
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

func (a *azureActions) WriteToStream(ctx context.Context, writer io.WriteCloser) error {
	resources, err := a.GroupResourceList(ctx)
	go a.ResourcesList(ctx, resources, writer)
	return err
}
