package install

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/rhcos"
	"github.com/openshift/installer/pkg/asset/targets"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/arm"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

var apiVersions = map[string]string{
	"authorization": "2015-07-01",
	"compute":       "2019-03-01",
	"msi":           "2018-11-30",
	"network":       "2019-07-01",
	"privatedns":    "2018-09-01",
	"storage":       "2019-04-01",
}

func (i *Installer) installStorage(ctx context.Context, oc *api.OpenShiftCluster, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds) error {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return err
	}

	image := &releaseimage.Image{
		// https://openshift-release.svc.ci.openshift.org/
		// oc adm release info quay.io/openshift-release-dev/ocp-release-nightly:4.3.0-0.nightly-2019-11-19-122017
		PullSpec:   "quay.io/openshift-release-dev/ocp-release-nightly@sha256:ab5022516a948e40190e4ce5729737780b96c96d2cf4d3fc665105b32d751d20",
		Repository: "quay.io/openshift-release-dev/ocp-release-nightly",

		// oc adm release info quay.io/openshift-release-dev/ocp-release:4.2.4
		// PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:cebce35c054f1fb066a4dc0a518064945087ac1f3637fe23d2ee2b0c433d6ba8",
		// Repository: "quay.io/openshift-release-dev/ocp-release",
	}

	clusterID := &installconfig.ClusterID{
		UUID:    uuid.NewV4().String(),
		InfraID: oc.Name,
	}

	g := graph{
		reflect.TypeOf(installConfig): installConfig,
		reflect.TypeOf(platformCreds): platformCreds,
		reflect.TypeOf(image):         image,
		reflect.TypeOf(clusterID):     clusterID,
	}

	for _, a := range targets.Cluster {
		_, err := g.resolve(a)
		if err != nil {
			return err
		}
	}

	adminClient := g[reflect.TypeOf(&kubeconfig.AdminClient{})].(*kubeconfig.AdminClient)
	bootstrap := g[reflect.TypeOf(&bootstrap.Bootstrap{})].(*bootstrap.Bootstrap)
	rhcosImage := g[reflect.TypeOf(new(rhcos.Image))].(*rhcos.Image)

	i.log.Print("creating resource group")
	_, err = i.groups.CreateOrUpdate(ctx, oc.Properties.ResourceGroup, resources.Group{
		Location: &installConfig.Config.Azure.Region,
	})
	if err != nil {
		return err
	}

	{
		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources: []arm.Resource{
				{
					// deploy the Identity now to give AAD a chance to update
					// itself before we apply the RBAC rule in the next
					// deployment
					Resource: &msi.Identity{
						Name:     to.StringPtr(clusterID.InfraID + "-identity"),
						Location: &installConfig.Config.Azure.Region,
						Type:     "Microsoft.ManagedIdentity/userAssignedIdentities",
					},
					APIVersion: apiVersions["msi"],
				},
				{
					Resource: &storage.Account{
						Sku: &storage.Sku{
							Name: "Standard_LRS",
						},
						Name:     to.StringPtr("cluster" + oc.Properties.StorageSuffix),
						Location: &installConfig.Config.Azure.Region,
						Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					},
					APIVersion: apiVersions["storage"],
				},
				{
					// should go away when we use a cloud partner image
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + oc.Properties.StorageSuffix + "/default/vhd"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + oc.Properties.StorageSuffix,
					},
				},
				{
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + oc.Properties.StorageSuffix + "/default/ignition"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + oc.Properties.StorageSuffix,
					},
				},
				{
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + oc.Properties.StorageSuffix + "/default/aro"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + oc.Properties.StorageSuffix,
					},
				},
				{
					Resource: &network.SecurityGroup{
						SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
							SecurityRules: &[]network.SecurityRule{
								{
									SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
										Protocol:                 network.SecurityRuleProtocolTCP,
										SourcePortRange:          to.StringPtr("*"),
										DestinationPortRange:     to.StringPtr("6443"),
										SourceAddressPrefix:      to.StringPtr("*"),
										DestinationAddressPrefix: to.StringPtr("*"),
										Access:                   network.SecurityRuleAccessAllow,
										Priority:                 to.Int32Ptr(101),
										Direction:                network.SecurityRuleDirectionInbound,
									},
									Name: to.StringPtr("apiserver_in"),
								},
								{
									SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
										Protocol:                 network.SecurityRuleProtocolTCP,
										SourcePortRange:          to.StringPtr("*"),
										DestinationPortRange:     to.StringPtr("22"),
										SourceAddressPrefix:      to.StringPtr("*"),
										DestinationAddressPrefix: to.StringPtr("*"),
										Access:                   network.SecurityRuleAccessAllow,
										Priority:                 to.Int32Ptr(103),
										Direction:                network.SecurityRuleDirectionInbound,
									},
									Name: to.StringPtr("bootstrap_ssh_in"),
								},
							},
						},
						Name:     to.StringPtr(clusterID.InfraID + "-controlplane-nsg"),
						Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
				{
					Resource: &network.SecurityGroup{
						Name:     to.StringPtr(clusterID.InfraID + "-node-nsg"),
						Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
						Location: &installConfig.Config.Azure.Region,
					},
					APIVersion: apiVersions["network"],
				},
			},
		}

		i.log.Print("deploying storage template")
		future, err := i.deployments.CreateOrUpdate(ctx, oc.Properties.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: t,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			return err
		}

		i.log.Print("waiting for storage template deployment")
		err = future.WaitForCompletionRef(ctx, i.deployments.Client)
		if err != nil {
			return err
		}
	}

	{
		blobService, err := i.getBlobService(ctx, oc)
		if err != nil {
			return err
		}

		// blob copying should go away when we use a cloud partner image
		i.log.Print("copying rhcos blob")
		rhcosVhd := blobService.GetContainerReference("vhd").GetBlobReference("rhcos" + oc.Properties.StorageSuffix + ".vhd")
		err = rhcosVhd.Copy(string(*rhcosImage), nil)
		if err != nil {
			return err
		}

		rhcosVhd.Metadata = azstorage.BlobMetadata{
			"source_uri": "var.azure_image_url", // https://github.com/openshift/installer/pull/2468
		}

		err = rhcosVhd.SetMetadata(nil)
		if err != nil {
			return err
		}

		bootstrapIgn := blobService.GetContainerReference("ignition").GetBlobReference("bootstrap.ign")
		err = bootstrapIgn.CreateBlockBlobFromReader(bytes.NewReader(bootstrap.File.Data), nil)
		if err != nil {
			return err
		}

		// the graph is quite big so we store it in a storage account instead of
		// in cosmosdb
		graph := blobService.GetContainerReference("aro").GetBlobReference("graph")
		b, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			return err
		}

		err = graph.CreateBlockBlobFromReader(bytes.NewReader(b), nil)
		if err != nil {
			return err
		}
	}

	for subnetID, nsgID := range map[string]string{
		oc.Properties.MasterProfile.SubnetID:     "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + clusterID.InfraID + "-controlplane-nsg",
		oc.Properties.WorkerProfiles[0].SubnetID: "/subscriptions/" + r.SubscriptionID + "/resourceGroups/" + oc.Properties.ResourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + clusterID.InfraID + "-node-nsg",
	} {
		i.log.Printf("attaching network security group to subnet %s", subnetID)

		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := subnet.Get(ctx, &oc.Properties.ServicePrincipalProfile, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat == nil {
			s.SubnetPropertiesFormat = &network.SubnetPropertiesFormat{}
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = &network.SecurityGroup{
			ID: to.StringPtr(nsgID),
		}

		err = subnet.CreateOrUpdate(ctx, &oc.Properties.ServicePrincipalProfile, subnetID, s)
		if err != nil {
			return err
		}
	}

	{
		identity, err := i.userassignedidentities.Get(ctx, oc.Properties.ResourceGroup, clusterID.InfraID+"-identity")
		if err != nil {
			return err
		}

		_, err = i.roleassignments.Create(ctx, "/subscriptions/"+r.SubscriptionID+"/resourceGroups/"+installConfig.Config.Azure.NetworkResourceGroupName+"/providers/Microsoft.Network/virtualNetworks/"+installConfig.Config.Azure.VirtualNetwork, uuid.NewV4().String(), authorization.RoleAssignmentCreateParameters{
			Properties: &authorization.RoleAssignmentProperties{
				RoleDefinitionID: to.StringPtr("/subscriptions/" + r.SubscriptionID + "/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"), // Owner
				PrincipalID:      to.StringPtr(identity.PrincipalID.String()),
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = i.db.Patch(oc.ID, func(doc *api.OpenShiftClusterDocument) (err error) {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		doc.OpenShiftCluster.Properties.Installation.Now = time.Now().UTC()
		doc.OpenShiftCluster.Properties.ClusterID = clusterID.InfraID
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminClient.File.Data
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
