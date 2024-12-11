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
	secGroups, err := rc.securitygroupscli.List(ctx, *resourceGroup.Name, nil)
	if err != nil {
		return err
	}

	for _, secGroup := range secGroups {
		if secGroup.Properties == nil || secGroup.Properties.Subnets == nil {
			continue
		}

		vnetID, _, err := apisubnet.Split(*subnet.ID)
		if err != nil {
			return err
		}

		r, err := azure.ParseResourceID(vnetID)
		if err != nil {
			return err
		}

		for _, secGroupSubnet := range secGroup.Properties.Subnets {
			subnet, err := rc.subnet.Get(ctx, *resourceGroup.Name, r.ResourceName, *secGroupSubnet.Name, nil)
			if err != nil {
				return err
			}

			rc.log.Debugf("Removing security group from subnet: %s/%s/%s", *resourceGroup.Name, *secGroup.Name, *subnet.Name)

			if !rc.dryRun {
				if subnet.Properties.NetworkSecurityGroup == nil {
					continue
				}

				subnet.Properties.NetworkSecurityGroup = nil

				vnetID, _, err := apisubnet.Split(*subnet.ID)
				if err != nil {
					return err
				}

				r, err := azure.ParseResourceID(vnetID)
				if err != nil {
					return err
				}
				err = rc.subnet.CreateOrUpdateAndWait(ctx, *resourceGroup.Name, r.ResourceName, *subnet.Name, subnet.Subnet, nil)
				rc.log.Debugf("Checking VNet name: %s - %s - %s", *resourceGroup.Name, r.ResourceName, *subnet.Name)
				if err != nil {
					return err
				}
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
