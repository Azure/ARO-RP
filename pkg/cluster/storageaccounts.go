package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// enableServiceEndpoints should enable service endpoints on
// subnets for storage account access
func (m *manager) enableServiceEndpoints(ctx context.Context) error {
	subnets := []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
	}

	for _, wp := range m.doc.OpenShiftCluster.Properties.WorkerProfiles {
		subnets = append(subnets, wp.SubnetID)
	}

	for _, subnetId := range subnets {
		subnet, err := m.subnet.Get(ctx, subnetId)
		if err != nil {
			return err
		}

		var changed bool
		for _, endpoint := range api.SubnetsEndpoints {
			var found bool
			if subnet != nil && subnet.ServiceEndpoints != nil {
				for _, se := range *subnet.ServiceEndpoints {
					if strings.EqualFold(*se.Service, endpoint) &&
						se.ProvisioningState == mgmtnetwork.Succeeded {
						found = true
					}
				}
			}
			if !found {
				if subnet.ServiceEndpoints == nil {
					subnet.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{}
				}
				*subnet.ServiceEndpoints = append(*subnet.ServiceEndpoints, mgmtnetwork.ServiceEndpointPropertiesFormat{
					Service:   to.StringPtr(endpoint),
					Locations: &[]string{"*"},
				})
				changed = true
			}
		}
		if changed {
			err := m.subnet.CreateOrUpdate(ctx, subnetId, subnet)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// migrateStorageAccounts redeploys storage accounts with firewall rules preventing external access
// The encryption flag is set to false/disabled for legacy storage accounts.
func (m *manager) migrateStorageAccounts(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix
	registryStorageAccountName := m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			m.storageAccount(clusterStorageAccountName, m.doc.OpenShiftCluster.Location, false),
			m.storageAccount(registryStorageAccountName, m.doc.OpenShiftCluster.Location, false),
		},
	}

	return m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
}

func (m *manager) populateRegistryStorageAccountName(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName != "" {
		return nil
	}

	rc, err := m.imageregistrycli.ImageregistryV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = rc.Spec.Storage.Azure.AccountName
		return nil
	})
	return err
}
