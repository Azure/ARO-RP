package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var apiVersions = map[string]string{
	"authorization": "2015-07-01",
	"compute":       "2019-03-01",
	"network":       "2019-07-01",
	"privatedns":    "2018-09-01",
	"storage":       "2019-04-01",
}

func (i *Installer) createDNS(ctx context.Context) error {
	err := i.dns.Create(ctx, i.doc.OpenShiftCluster)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		strings.Contains(detailedErr.Message, "already registered") {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeDuplicateDomain, "", "The provided domain '%s' is already in use by a cluster.", err)
	}
	return err
}

func (i *Installer) installStorage(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	clusterID := &installconfig.ClusterID{
		UUID:    uuid.NewV4().String(),
		InfraID: "aro",
	}

	g := graph{
		reflect.TypeOf(installConfig): installConfig,
		reflect.TypeOf(platformCreds): platformCreds,
		reflect.TypeOf(image):         image,
		reflect.TypeOf(clusterID):     clusterID,
	}

	i.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		_, err := g.resolve(a)
		if err != nil {
			return err
		}
	}

	adminClient := g[reflect.TypeOf(&kubeconfig.AdminClient{})].(*kubeconfig.AdminClient)

	resourceGroup := i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')+1:]

	i.log.Print("creating resource group")
	group := mgmtresources.Group{
		Location:  &installConfig.Config.Azure.Region,
		ManagedBy: to.StringPtr(i.doc.OpenShiftCluster.ID),
	}
	if _, ok := i.env.(env.Dev); ok {
		group.ManagedBy = nil
	}
	_, err := i.groups.CreateOrUpdate(ctx, resourceGroup, group)
	if err != nil {
		return err
	}

	if development, ok := i.env.(env.Dev); ok {
		err = development.CreateARMResourceGroupRoleAssignment(ctx, i.fpAuthorizer, resourceGroup)
		if err != nil {
			return err
		}
	}

	{
		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources: []*arm.Resource{
				{
					Resource: &mgmtstorage.Account{
						Sku: &mgmtstorage.Sku{
							Name: "Standard_LRS",
						},
						Name:     to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix),
						Location: &installConfig.Config.Azure.Region,
						Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					},
					APIVersion: apiVersions["storage"],
				},
				{
					Resource: &mgmtstorage.BlobContainer{
						Name: to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/ignition"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix,
					},
				},
				{
					Resource: &mgmtstorage.BlobContainer{
						Name: to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/aro"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix,
					},
				},
				{
					Resource: &mgmtnetwork.SecurityGroup{
						SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
							SecurityRules: &[]mgmtnetwork.SecurityRule{
								{
									SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
										Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
										SourcePortRange:          to.StringPtr("*"),
										DestinationPortRange:     to.StringPtr("6443"),
										SourceAddressPrefix:      to.StringPtr("*"),
										DestinationAddressPrefix: to.StringPtr("*"),
										Access:                   mgmtnetwork.SecurityRuleAccessAllow,
										Priority:                 to.Int32Ptr(101),
										Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
									},
									Name: to.StringPtr("apiserver_in"),
								},
								{
									SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
										Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
										SourcePortRange:          to.StringPtr("*"),
										DestinationPortRange:     to.StringPtr("22"),
										SourceAddressPrefix:      to.StringPtr("*"),
										DestinationAddressPrefix: to.StringPtr("*"),
										Access:                   mgmtnetwork.SecurityRuleAccessAllow,
										Priority:                 to.Int32Ptr(103),
										Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
									},
									Name: to.StringPtr("bootstrap_ssh_in"),
								},
							},
						},
						Name:     to.StringPtr("aro-controlplane-nsg"),
						Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
				{
					Resource: &mgmtnetwork.SecurityGroup{
						Name:     to.StringPtr("aro-node-nsg"),
						Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
			},
		}

		i.log.Print("deploying storage template")
		err = i.deployments.CreateOrUpdateAndWait(ctx, resourceGroup, "azuredeploy", mgmtresources.Deployment{
			Properties: &mgmtresources.DeploymentProperties{
				Template: t,
				Mode:     mgmtresources.Incremental,
			},
		})
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); ok {
				if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
					requestErr.ServiceError != nil &&
					requestErr.ServiceError.Code == "DeploymentActive" {
					i.log.Print("waiting for storage template")
					err = i.deployments.Wait(ctx, resourceGroup, "azuredeploy")
				}
			}
			if err != nil {
				return err
			}
		}
	}

	{

		// the graph is quite big so we store it in a storage account instead of
		// in cosmosdb
		err := i.saveGraph(ctx, g)
		if err != nil {
			return err
		}
	}

	for _, subnetID := range []string{
		i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		i.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	} {
		i.log.Printf("attaching network security group to subnet %s", subnetID)

		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := i.subnet.Get(ctx, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat == nil {
			s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
		}

		nsgID, err := subnet.NetworkSecurityGroupID(i.doc.OpenShiftCluster, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			if strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
				continue
			}

			return fmt.Errorf("tried to overwrite non-nil network security group")
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: to.StringPtr(nsgID),
		}

		err = i.subnet.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			return err
		}
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		doc.OpenShiftCluster.Properties.Install.Now = time.Now().UTC()
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminClient.File.Data
		return nil
	})
	return err
}
