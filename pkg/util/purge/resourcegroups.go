package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"golang.org/x/sync/errgroup"
)

const maxConcurrentResourceGroupOperations = 5

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
	var resourceGroupsToDelete []mgmtfeatures.ResourceGroup
	for _, g := range gs {
		if !rc.shouldDelete(g, rc.log) {
			continue
		}

		if rc.dryRun {
			rc.log.Printf("DRY-RUN: Would delete ResourceGroup: %s", *g.Name)
			continue
		}

		resourceGroupsToDelete = append(resourceGroupsToDelete, g)
	}

	resourceGroupsToDelete = runResourceGroupTasks(resourceGroupsToDelete, func(g mgmtfeatures.ResourceGroup) error {
		err := rc.cleanResourceGroupResources(ctx, g)
		if err != nil {
			rc.log.Errorf("error cleaning resource group %v: %v", *g.Name, err)
		}
		return err
	})

	runResourceGroupTasks(resourceGroupsToDelete, func(g mgmtfeatures.ResourceGroup) error {
		err := rc.resourcegroupscli.DeleteAndWait(ctx, *g.Name)
		if err != nil {
			rc.log.Errorf("error deleting resource group %v: %v", *g.Name, err)
			return err
		}
		rc.log.Printf("Deleted ResourceGroup: %s", *g.Name)
		return nil
	})

	return nil
}

func runResourceGroupTasks(resourceGroups []mgmtfeatures.ResourceGroup, task func(mgmtfeatures.ResourceGroup) error) []mgmtfeatures.ResourceGroup {
	var group errgroup.Group
	group.SetLimit(maxConcurrentResourceGroupOperations)
	var mu sync.Mutex
	succeeded := make([]mgmtfeatures.ResourceGroup, 0, len(resourceGroups))

	for _, g := range resourceGroups {
		group.Go(func() error {
			if err := task(g); err != nil {
				return nil
			}
			mu.Lock()
			succeeded = append(succeeded, g)
			mu.Unlock()
			return nil
		})
	}
	_ = group.Wait()

	return succeeded
}

// cleanResourceGroupResources prepares the group for deletion:
//   - unassign subnets
//   - clean private links
func (rc *ResourceCleaner) cleanResourceGroupResources(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	rc.log.Printf("Deleting ResourceGroup: %s", *resourceGroup.Name)
	err := rc.cleanNetworking(ctx, resourceGroup)
	if err != nil {
		return err
	}

	err = rc.cleanPrivateLink(ctx, resourceGroup)
	if err != nil {
		return err
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
