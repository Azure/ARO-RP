package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (r *reconcileManager) reconcileAccounts(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(r.instance.Spec.ClusterResourceGroupID, '/')

	subnets, err := r.kubeSubnets.List(ctx)
	if err != nil {
		return err
	}

	serviceSubnets := r.instance.Spec.ServiceSubnets
	for _, subnet := range subnets {
		serviceSubnets = append(serviceSubnets, subnet.ResourceID)
	}

	rc, err := r.imageregistrycli.ImageregistryV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	storageAccounts := []string{
		"cluster" + r.instance.Spec.StorageSuffix, // this is our creation, so name is deterministic
		rc.Spec.Storage.Azure.AccountName,
	}

	for _, accountName := range storageAccounts {
		var changed bool

		account, err := r.storage.GetProperties(ctx, resourceGroup, accountName, "")
		if err != nil {
			return err
		}

		for _, subnet := range serviceSubnets {
			// if subnet ResourceID was found and we need to append
			found := false

			if account.AccountProperties.NetworkRuleSet != nil && account.AccountProperties.NetworkRuleSet.VirtualNetworkRules != nil {
				for _, rule := range *account.AccountProperties.NetworkRuleSet.VirtualNetworkRules {
					if strings.EqualFold(to.String(rule.VirtualNetworkResourceID), subnet) {
						found = true
						break
					}
				}
			}

			// if rule was not found - we add it
			if !found {
				*account.AccountProperties.NetworkRuleSet.VirtualNetworkRules = append(*account.AccountProperties.NetworkRuleSet.VirtualNetworkRules, mgmtstorage.VirtualNetworkRule{
					VirtualNetworkResourceID: to.StringPtr(subnet),
					Action:                   mgmtstorage.Allow,
				})
				changed = true
			}
		}

		if changed {
			sa := mgmtstorage.AccountUpdateParameters{
				AccountPropertiesUpdateParameters: &mgmtstorage.AccountPropertiesUpdateParameters{
					NetworkRuleSet: account.AccountProperties.NetworkRuleSet,
				},
			}

			_, err = r.storage.Update(ctx, resourceGroup, accountName, sa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
