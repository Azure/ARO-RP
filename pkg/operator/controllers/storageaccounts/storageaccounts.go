package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type manager interface {
	checkClusterSubnetsToReconcile(ctx context.Context, clusterSubnets []string) ([]string, error)
	reconcileAccounts(ctx context.Context, subnets []string, storageAccounts []string) error
}

type newManager func(
	log *logrus.Entry,
	location, subscriptionID, resourceGroup string,
	azenv azureclient.AROEnvironment, authorizer autorest.Authorizer,
) manager

// reconcileManager is instance of manager instantiated per request
type reconcileManager struct {
	log *logrus.Entry

	location, resourceGroup string

	subnet  subnet.Manager
	storage storage.AccountsClient
}

func newReconcileManager(
	log *logrus.Entry,

	location, subscriptionID, resourceGroup string,

	azenv azureclient.AROEnvironment,
	authorizer autorest.Authorizer,
) manager {
	return &reconcileManager{
		log: log,

		location:      location,
		resourceGroup: resourceGroup,

		subnet:  subnet.NewManager(&azenv, subscriptionID, authorizer),
		storage: storage.NewAccountsClient(&azenv, subscriptionID, authorizer),
	}
}

// checkClusterSubnetsToReconcile will check cluster subnets for the Microsoft.Storage service endpoint.
// If the subnet has the service endpoint, it needs to be included in the storage account vnet rules.
func (r *reconcileManager) checkClusterSubnetsToReconcile(ctx context.Context, clusterSubnets []string) ([]string, error) {
	subnetsToReconcile := []string{}

	for _, subnet := range clusterSubnets {
		mgmtSubnet, err := r.subnet.Get(ctx, subnet)
		if err != nil {
			if azureerrors.IsNotFoundError(err) {
				r.log.Infof("Subnet %s not found, skipping", subnet)
				break
			}
			return nil, err
		}

		if mgmtSubnet.SubnetPropertiesFormat != nil && mgmtSubnet.SubnetPropertiesFormat.ServiceEndpoints != nil {
			for _, serviceEndpoint := range *mgmtSubnet.SubnetPropertiesFormat.ServiceEndpoints {
				isStorageEndpoint := (serviceEndpoint.Service != nil) && (*serviceEndpoint.Service == "Microsoft.Storage")
				matchesClusterLocation := false

				if serviceEndpoint.Locations != nil {
					for _, l := range *serviceEndpoint.Locations {
						if l == "*" || l == r.location {
							matchesClusterLocation = true
							break
						}
					}
				}

				if isStorageEndpoint && matchesClusterLocation {
					subnetsToReconcile = append(subnetsToReconcile, subnet)
					break
				}
			}
		}
	}

	return subnetsToReconcile, nil
}

func (r *reconcileManager) reconcileAccounts(ctx context.Context, subnets, storageAccounts []string) error {
	for _, accountName := range storageAccounts {
		var changed bool

		account, err := r.storage.GetProperties(ctx, r.resourceGroup, accountName, "")
		if err != nil {
			return err
		}

		for _, subnet := range subnets {
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

			_, err = r.storage.Update(ctx, r.resourceGroup, accountName, sa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
