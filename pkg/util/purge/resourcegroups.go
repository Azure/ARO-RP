package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
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
