package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

func (rc *ResourceCleaner) PruneOldDevSubnets(ctx context.Context, resourceGroupName string, withNSGs bool) error {
	vnetName := "dev-vnet"
	vnetUsages, err := rc.vnetcli.GetUsage(ctx, resourceGroupName, vnetName, nil)
	if err != nil {
		return err
	}

	for _, v := range vnetUsages {
		if *v.CurrentValue == 0 {
			r, err := arm.ParseResourceID(*v.ID)
			if err != nil {
				return err
			}

			// only include ones made for clusters in dev env
			if !(strings.HasSuffix(r.Name, "-master") || strings.HasSuffix(r.Name, "-worker")) {
				continue
			}

			rc.log.Printf("Fetching subnet %s/%s", r.Parent.Name, r.Name)

			subnet, err := rc.subnet.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
			if err != nil {
				return err
			}

			if subnet.Properties.NetworkSecurityGroup != nil {
				if !withNSGs {
					rc.log.Printf("subnet %s/%s has NSG, skipping", r.Parent.Name, r.Name)
					continue
				}
				if !rc.dryRun {
					rc.log.Printf("[DRY-RUN=False] Resources Detaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", resourceGroupName, subnet.Properties.NetworkSecurityGroup.Name, r.ResourceGroupName, r.Parent.Name, r.Name)
					subnet.Properties.NetworkSecurityGroup = nil

					err = rc.subnet.CreateOrUpdateAndWait(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, subnet.Subnet, nil)
					if err != nil {
						return err
					}
				} else {
					rc.log.Printf("[DRY-RUN=True] Resources Detaching: NSG RG: %s - NSG: %v || Subnet RG: %v vetName.ResourceName: %s - subnetName: %s", resourceGroupName, subnet.Properties.NetworkSecurityGroup.Name, r.ResourceGroupName, r.Parent.Name, r.Name)
				}
			}

			if subnet.Properties.RouteTable != nil {
				rtRID, err := arm.ParseResourceID(*subnet.Properties.RouteTable.ID)
				if err != nil {
					return err
				}

				if !rc.dryRun {
					rc.log.Printf("[DRY-RUN=False] Resources Detaching: Route Table RG: %s - RT: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", rtRID.ResourceGroupName, rtRID.Name, r.ResourceGroupName, r.Parent.Name, r.Name)

					subnet.Properties.RouteTable = nil
					err = rc.subnet.CreateOrUpdateAndWait(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, subnet.Subnet, nil)
					if err != nil {
						return err
					}
				} else {
					rc.log.Printf("[DRY-RUN=True] Resources Detaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", rtRID.ResourceGroupName, rtRID.Name, r.ResourceGroupName, r.Parent.Name, r.Name)
				}
			}
			if !rc.dryRun {
				rc.log.Printf("[DRY-RUN=False] Removing subnet %s/%s", r.Parent.Name, r.Name)
				err = rc.subnet.DeleteAndWait(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
				if err != nil {
					return err
				}
			} else {
				rc.log.Printf("[DRY-RUN=True] Removing subnet %s/%s", r.Parent.Name, r.Name)
			}
		}
	}

	return nil
}

// cleanNetworking lists subnets in vnets and unnassign security groups
func (rc *ResourceCleaner) cleanNetworking(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	networkSecurityGroups, err := rc.securitygroupscli.List(ctx, *resourceGroup.Name, nil)
	if err != nil {
		return err
	}

	for _, networkSecGroup := range networkSecurityGroups {
		if networkSecGroup.Properties == nil || networkSecGroup.Properties.Subnets == nil {
			continue
		}

		for _, nsgSubnet := range networkSecGroup.Properties.Subnets {
			r, err := arm.ParseResourceID(*nsgSubnet.ID)
			if err != nil {
				return err
			}
			subnet, err := rc.subnet.Get(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, nil)
			if err != nil {
				return err
			}

			rc.log.Printf("Dettaching NSG from subnet: %s/%s/%s", *resourceGroup.Name, *networkSecGroup.Name, *subnet.Name)

			if !rc.dryRun {
				if subnet.Properties.NetworkSecurityGroup == nil {
					continue
				}

				subnet.Properties.NetworkSecurityGroup = nil

				err = rc.subnet.CreateOrUpdateAndWait(ctx, r.ResourceGroupName, r.Parent.Name, r.Name, subnet.Subnet, nil)
				if err != nil {
					return err
				}
				rc.log.Printf("[DRY-RUN=False] Resources Dettaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", *resourceGroup.Name, *networkSecGroup.Name, r.ResourceGroupName, r.Parent.Name, r.Name)
			} else {
				rc.log.Printf("[DRY-RUN=True] Resources Dettaching: NSG RG: %s - NSG: %v || Subnet RG: %v vnetName.ResourceName: %s - subnetName: %s", *resourceGroup.Name, *networkSecGroup.Name, r.ResourceGroupName, r.Parent.Name, r.Name)
			}
		}
	}

	return nil
}

// cleanPrivateLink lists and unassigns all private links. If they are assigned the deletion will fail
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
