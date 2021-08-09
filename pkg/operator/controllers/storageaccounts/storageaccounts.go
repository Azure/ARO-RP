package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (r *reconcileManager) reconcileAccounts(ctx context.Context) error {

	subnets, err := r.subnets.ListFromCluster(ctx)
	if err != nil {
		return err
	}

	resourceGroup := stringutils.LastTokenByte(r.instance.Spec.ClusterResourceGroupID, '/')

	storageAccounts := []string{
		"imageregistry" + r.instance.Spec.StorageSuffix,
		"cluster" + r.instance.Spec.StorageSuffix,
	}

	for _, accountName := range storageAccounts {
		var changed bool

		account, err := r.storage.GetProperties(ctx, resourceGroup, accountName, "")
		if err != nil {
			return err
		}

		for _, subnet := range subnets {
			// if subnet.ID was found and we need to append
			found := false

			if account.NetworkRuleSet != nil && account.NetworkRuleSet.VirtualNetworkRules != nil {
				for _, rule := range *account.NetworkRuleSet.VirtualNetworkRules {
					// Name is confusing. In fact this is Subnet ResourceID...
					if strings.EqualFold(to.String(rule.VirtualNetworkResourceID), subnet.ResourceID) {
						found = true
						break
					}
				}
			}

			// if rule was not found - we add it
			if !found {
				*account.NetworkRuleSet.VirtualNetworkRules = append(*account.NetworkRuleSet.VirtualNetworkRules, mgmtstorage.VirtualNetworkRule{
					VirtualNetworkResourceID: to.StringPtr(subnet.ResourceID),
					Action:                   mgmtstorage.Allow,
				})
				changed = true
			}
		}

		if changed {
			sa := mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					NetworkRuleSet: account.NetworkRuleSet,
				},
			}
			account, err = r.storage.Update(ctx, resourceGroup, accountName, sa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
