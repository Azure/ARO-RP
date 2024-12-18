package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
)

// CleanResourceGroups loop through the resourgroups in the subscription
// and deleted everything that is not marked for deletion
// The deletion check is performed by passed function: DeleteCheck
func (rc *ResourceCleaner) CleanResourceGroups(ctx context.Context) error {
	// every resource have to live in the group, therefore deletion clean the unused groups at first
	gs, err := rc.resourcegroupscli.List(ctx, "", nil)
	if err != nil {
		return err
	}

	sort.Slice(gs, func(i, j int) bool { return *gs[i].Name < *gs[j].Name })
	for _, g := range gs {
		err := rc.cleanResourceGroup(ctx, g)
		if err != nil {
			rc.log.Error(err)
		}
	}

	return nil
}

// cleanResourceGroup checkes whether the resource group can be deleted if yes proceed to clean the group in an order:
//   - unassign subnets
//   - clean private links
//   - checks ARO presence -> store app object ID for futher use
//   - deletes resource group
func (rc *ResourceCleaner) cleanResourceGroup(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	if rc.shouldDelete(resourceGroup, rc.log) {
		rc.log.Printf("Deleting ResourceGroup: %s", *resourceGroup.Name)
		err := rc.cleanNetworking(ctx, resourceGroup)
		if err != nil {
			return err
		}

		err = rc.cleanPrivateLink(ctx, resourceGroup)
		if err != nil {
			return err
		}

		if !rc.dryRun {
			_, err := rc.resourcegroupscli.Delete(ctx, *resourceGroup.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanNetworking lists subnets in vnets and unnassign security groups
func (rc *ResourceCleaner) cleanNetworking(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	nsGroups, err := rc.securitygroupscli.List(ctx, *resourceGroup.Name, nil)
	if err != nil {
		return err
	}
	rc.log.Printf(">>>> Entering 'cleanNetworking' RG: %s", *resourceGroup.Name)

	for _, networkSecurityGroup := range nsGroups {
		if networkSecurityGroup.Properties == nil || networkSecurityGroup.Properties.Subnets == nil {
			continue
		}
		rc.log.Printf(">>>> Entering 'secGroup' loop | *secGroup.ID: %s", *networkSecurityGroup.ID)

		for _, SubnetResource := range networkSecurityGroup.Properties.Subnets {
			rc.log.Printf("[FOR LOOP] What is SubnetResource.ID: %v", *SubnetResource.ID)

			vnetID, subnetName, err := apisubnet.Split(*SubnetResource.ID)
			if err != nil {
				return err
			}

			vnetName, err := azure.ParseResourceID(vnetID)
			if err != nil {
				return err
			}

			rc.log.Printf("Before 'GET' RG: %s - vnetName.ResourceName: %s - subnetName: %s - *secGroup.Name: %v", *resourceGroup.Name, vnetName.ResourceName, subnetName, *networkSecurityGroup.Name)

			subnet, err := rc.subnet.Get(ctx, *resourceGroup.Name, vnetName.ResourceName, subnetName, nil)
			if err != nil {
				return err
			}

			rc.log.Printf("After 'GET' subnet.Name: %v", subnet.Name)
			rc.log.Printf("After 'GET' *SubnetResource.ID': %v", *SubnetResource.ID)
			rc.log.Printf("After 'GET' subnet.Properties.NetworkSecurityGroup: %v", subnet.Properties.NetworkSecurityGroup)
			rc.log.Printf("After 'GET' RG: %s - vnetName.ResourceName: %s - subnetName: %s - *secGroup.Name: %v", *resourceGroup.Name, vnetName.ResourceName, subnetName, *networkSecurityGroup.Name)

			if !rc.dryRun {
				if subnet.Properties.NetworkSecurityGroup == nil {
					continue
				}

				subnet.Properties.NetworkSecurityGroup = nil
				rc.log.Printf("Resources Dettaching: RG: %s - Vnet: %s - secGroupSubnet: %s", *resourceGroup.Name, vnetName.ResourceName, *SubnetResource.Name)
				err = rc.subnet.CreateOrUpdateAndWait(ctx, *resourceGroup.Name, vnetName.ResourceName, *SubnetResource.Name, subnet.Subnet, nil)
				if err != nil {
					return err
				}
			} else {
				rc.log.Printf("Resources Dettaching: \nRG: %s - vnetName.ResourceName: %s - subnetName: %s - *secGroup.Name: %v", *resourceGroup.Name, vnetName.ResourceName, subnetName, *networkSecurityGroup.Name)
			}
		}
	}

	return nil
}

// cleanPrivateLink lists and unassigns all private links. If they are assigned the deletoin will fail
func (rc *ResourceCleaner) cleanPrivateLink(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	plss, err := rc.privatelinkservicescli.List(ctx, *resourceGroup.Name, nil)
	if err != nil {
		return err
	}
	for _, pls := range plss {
		if pls.Properties == nil || pls.Properties.PrivateEndpointConnections == nil {
			continue
		}

		for _, peconn := range pls.Properties.PrivateEndpointConnections {
			rc.log.Debugf("Deleting private endpoint connection %s/%s/%s", *resourceGroup.Name, *pls.Name, *peconn.Name)
			if rc.dryRun {
				continue
			}
			err := rc.privatelinkservicescli.DeletePrivateEndpointConnectionAndWait(ctx, *resourceGroup.Name, *pls.Name, *peconn.Name, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
