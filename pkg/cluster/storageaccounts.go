package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
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
		if len(wp.SubnetID) > 0 {
			subnets = append(subnets, wp.SubnetID)
		} else {
			return fmt.Errorf("WorkerProfile '%s' has no SubnetID; check that the corresponding MachineSet is valid", wp.Name)
		}
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
	if len(m.doc.OpenShiftCluster.Properties.WorkerProfiles) == 0 {
		m.log.Error("skipping migrateStorageAccounts due to missing WorkerProfiles.")
		return nil
	}
	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	imageRegistryStorageAccountName, err := m.imageRegistryStorageAccountName()
	if err != nil {
		return err
	}

	err = validateImageRegistryStorageAccountName(imageRegistryStorageAccountName)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			m.storageAccount(clusterStorageAccountName, m.doc.OpenShiftCluster.Location, false),
			m.storageAccount(imageRegistryStorageAccountName, m.doc.OpenShiftCluster.Location, false),
		},
	}

	return arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "storage", t, nil)
}

func (m *manager) populateRegistryStorageAccountName(ctx context.Context) error {
	imageRegistryStorageAccountName, err := m.imageRegistryStorageAccountName()
	if err != nil {
		return err
	}

	if imageRegistryStorageAccountName != "" {
		return nil
	}

	registryConfig, err := m.imageregistrycli.ImageregistryV1().Configs().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}

	accountNameMutator := getAccountNameMutator(ctx, registryConfig)

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, accountNameMutator)
	return err
}

// getAccountNameMutator returns a database.OpenShiftClusterDocumentMutator function
// that is responsible for mutating the image registry storage account name of an *api.OpenShiftClusterDocument
func getAccountNameMutator(ctx context.Context, registryConfig *imageregistryv1.Config) database.OpenShiftClusterDocumentMutator {
	return func(doc *api.OpenShiftClusterDocument) error {
		if doc == nil {
			return fmt.Errorf("OpenShift cluster document is nil")
		}

		if doc.OpenShiftCluster == nil {
			return fmt.Errorf("OpenShiftCluster info from OpenShift cluster document is nil")
		}

		imageRegistryStorageAccountName, err := getAccountName(registryConfig)
		if err != nil {
			return err
		}

		err = validateImageRegistryStorageAccountName(imageRegistryStorageAccountName)
		if err != nil {
			return err
		}

		doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName = imageRegistryStorageAccountName
		return nil
	}
}

func getAccountName(registryConfig *imageregistryv1.Config) (string, error) {
	if registryConfig == nil {
		return "", fmt.Errorf("image registry config is nil")
	}

	if registryConfig.Spec.Storage.Azure == nil {
		return "", fmt.Errorf("azure storage field is nil in image registry config")
	}

	return registryConfig.Spec.Storage.Azure.AccountName, nil
}

func validateImageRegistryStorageAccountName(imageRegistryStorageAccountName string) error {
	if imageRegistryStorageAccountName == "" {
		return fmt.Errorf("the cluster's image registry name is empty")
	}

	return nil
}
