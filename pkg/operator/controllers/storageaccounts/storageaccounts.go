package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	imageregistryv1 "github.com/openshift/api/imageregistry/v1"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (r *reconcileManager) reconcileAccounts(ctx context.Context) error {
	location := r.instance.Spec.Location
	resourceGroup := stringutils.LastTokenByte(r.instance.Spec.ClusterResourceGroupID, '/')

	serviceSubnets := r.instance.Spec.ServiceSubnets

	subnets, err := r.kubeSubnets.List(ctx)
	if err != nil {
		return err
	}

	// Check each of the cluster subnets for the Microsoft.Storage service endpoint. If the subnet has
	// the service endpoint, it needs to be included in the storage account vnet rules.
	for _, subnet := range subnets {
		mgmtSubnet, err := r.subnets.Get(ctx, subnet.ResourceID)
		if err != nil {
			if azureerrors.IsNotFoundError(err) {
				r.log.Infof("Subnet %s not found, skipping", subnet.ResourceID)
				break
			}
			return err
		}

		if mgmtSubnet.SubnetPropertiesFormat != nil && mgmtSubnet.ServiceEndpoints != nil {
			for _, serviceEndpoint := range *mgmtSubnet.ServiceEndpoints {
				isStorageEndpoint := (serviceEndpoint.Service != nil) && (*serviceEndpoint.Service == "Microsoft.Storage")
				matchesClusterLocation := false

				if serviceEndpoint.Locations != nil {
					for _, l := range *serviceEndpoint.Locations {
						if l == "*" || l == location {
							matchesClusterLocation = true
							break
						}
					}
				}

				if isStorageEndpoint && matchesClusterLocation {
					serviceSubnets = append(serviceSubnets, subnet.ResourceID)
					break
				}
			}
		}
	}

	rc := &imageregistryv1.Config{}
	err = r.client.Get(ctx, types.NamespacedName{Name: "cluster"}, rc)
	if err != nil {
		return err
	}

	if rc.Spec.Storage.Azure == nil {
		return fmt.Errorf("azure storage field is nil in image registry config")
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

			if account.NetworkRuleSet != nil && account.NetworkRuleSet.VirtualNetworkRules != nil {
				for _, rule := range *account.NetworkRuleSet.VirtualNetworkRules {
					if strings.EqualFold(to.String(rule.VirtualNetworkResourceID), subnet) {
						found = true
						break
					}
				}
			}

			// if rule was not found - we add it
			if !found {
				*account.NetworkRuleSet.VirtualNetworkRules = append(*account.NetworkRuleSet.VirtualNetworkRules, mgmtstorage.VirtualNetworkRule{
					VirtualNetworkResourceID: to.StringPtr(subnet),
					Action:                   mgmtstorage.ActionAllow,
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

			_, err = r.storage.Update(ctx, resourceGroup, accountName, sa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
