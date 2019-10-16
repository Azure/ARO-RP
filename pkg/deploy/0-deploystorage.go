package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/rhcos"
	"github.com/openshift/installer/pkg/asset/targets"

	"github.com/jim-minter/rp/pkg/api"
)

func (d *Deployer) deployStorage(ctx context.Context, doc *api.OpenShiftClusterDocument, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds) error {
	g := graph{
		reflect.TypeOf(installConfig): installConfig,
		reflect.TypeOf(platformCreds): platformCreds,
	}

	for _, a := range targets.Cluster {
		_, err := g.resolve(a)
		if err != nil {
			return err
		}
	}

	bootstrap := g[reflect.TypeOf(&bootstrap.Bootstrap{})].(*bootstrap.Bootstrap)
	clusterID := g[reflect.TypeOf(&installconfig.ClusterID{})].(*installconfig.ClusterID)
	rhcosImage := g[reflect.TypeOf(new(rhcos.Image))].(*rhcos.Image)

	d.log.Print("creating resource group")
	_, err := d.groups.CreateOrUpdate(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, resources.Group{
		Location: &installConfig.Config.Azure.Region,
	})
	if err != nil {
		return err
	}

	{
		t := &Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources: []Resource{
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
						Name:     to.StringPtr("cluster" + doc.OpenShiftCluster.Properties.StorageSuffix),
						Location: &installConfig.Config.Azure.Region,
						Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
					},
					APIVersion: apiVersions["storage"],
				},
				{
					// should go away when we use a cloud partner image
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + "/default/vhd"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + doc.OpenShiftCluster.Properties.StorageSuffix,
					},
				},
				{
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + "/default/ignition"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + doc.OpenShiftCluster.Properties.StorageSuffix,
					},
				},
				{
					Resource: &storage.BlobContainer{
						Name: to.StringPtr("cluster" + doc.OpenShiftCluster.Properties.StorageSuffix + "/default/aro"),
						Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
					},
					APIVersion: apiVersions["storage"],
					DependsOn: []string{
						"Microsoft.Storage/storageAccounts/cluster" + doc.OpenShiftCluster.Properties.StorageSuffix,
					},
				},
			},
		}

		d.log.Print("deploying storage template")
		future, err := d.deployments.CreateOrUpdate(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: t,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			return err
		}

		d.log.Print("waiting for storage template deployment")
		err = future.WaitForCompletionRef(ctx, d.deployments.Client)
		if err != nil {
			return err
		}
	}

	{
		blobService, err := d.getBlobService(ctx, doc)
		if err != nil {
			return err
		}

		// blob copying should go away when we use a cloud partner image
		d.log.Print("copying rhcos blob")
		rhcosVhd := blobService.GetContainerReference("vhd").GetBlobReference("rhcos" + doc.OpenShiftCluster.Properties.StorageSuffix + ".vhd")
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

	doc, err = d.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) (err error) {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		doc.OpenShiftCluster.Properties.Installation.Now = time.Now().UTC()
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
