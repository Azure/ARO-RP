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
	netwSecurityGroups, err := rc.securitygroupscli.List(ctx, *resourceGroup.Name, nil)
	if err != nil {
		return err
	}

	for _, networkSecGroup := range netwSecurityGroups {
		if networkSecGroup.Properties == nil || networkSecGroup.Properties.Subnets == nil {
			continue
		}

		for _, nsgSubnet := range networkSecGroup.Properties.Subnets {

			vnetID, subnetName, err := apisubnet.Split(*nsgSubnet.ID)
			if err != nil {
				return err
			}

			vnetName, err := azure.ParseResourceID(vnetID)
			if err != nil {
				return err
			}

			subnetRGName, err := apisubnet.SplitRG(*nsgSubnet.ID)
			if err != nil {
				return err
			}

			subnet, err := rc.subnet.Get(ctx, subnetRGName, vnetName.ResourceName, subnetName, nil)
			if err != nil {
				return err
			}

			rc.log.Printf("Dettaching NSG from subnet: %s/%s/%s", *resourceGroup.Name, *networkSecGroup.Name, *subnet.Name)

			if !rc.dryRun {
				if subnet.Properties.NetworkSecurityGroup == nil {
					continue
				}

				subnet.Properties.NetworkSecurityGroup = nil

				err = rc.subnet.CreateOrUpdateAndWait(ctx, subnetRGName, vnetName.ResourceName, subnetName, subnet.Subnet, nil)
				if err != nil {
					return err
				}
				rc.log.Printf("[DRY-RUN=False] Resources Dettaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", *resourceGroup.Name, *networkSecGroup.Name, subnetRGName, vnetName.ResourceName, subnetName)
			} else {
				rc.log.Printf("[DRY-RUN=True] Resources Dettaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", *resourceGroup.Name, *networkSecGroup.Name, subnetRGName, vnetName.ResourceName, subnetName)
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
