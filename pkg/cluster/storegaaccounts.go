package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// enableStorageAccountEndpoints should enable Microsoft.Storage endpoints on
// subnets for storage account access
func (m *manager) enableStorageAccountEndpoints(ctx context.Context) error {
	//resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	subnets := []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	}
	endpoints := []string{
		"Microsoft.ContainerRegistry",
		"Microsoft.Storage",
	}

	for _, subnetId := range subnets {
		subnet, err := m.subnet.Get(ctx, subnetId)
		if err != nil {
			return err
		}
		for _, endpoint := range endpoints {

			var found bool
			if *subnet.ServiceEndpoints != nil {
				for _, se := range *subnet.ServiceEndpoints {
					if strings.EqualFold(*se.Service, endpoint) &&
						se.ProvisioningState == mgmtnetwork.Succeeded {
						found = true
					}
				}
			}
			if !found {
				*subnet.ServiceEndpoints = append(*subnet.ServiceEndpoints, mgmtnetwork.ServiceEndpointPropertiesFormat{
					Service:   to.StringPtr(endpoint),
					Locations: &[]string{"*"},
				})

				err := m.subnet.CreateOrUpdate(ctx, subnetId, subnet)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// fixStorageAccounts should re-deploy storage account with encryption,
// KindV2 and firewalls rules preventing external access.
func (m *manager) fixStorageAccounts(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, clusterStorageAccountName)
	if err != nil {
		return err
	}

	var installConfig *installconfig.InstallConfig
	err = pg.Get(&installConfig)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			m.storageAccount(clusterStorageAccountName, installConfig.Config.Azure.Region, false),
		},
	}

	return m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
}
