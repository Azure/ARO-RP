package install

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/util/azureclient/authorization"
	"github.com/jim-minter/rp/pkg/util/azureclient/dns"
	"github.com/jim-minter/rp/pkg/util/azureclient/network"
	"github.com/jim-minter/rp/pkg/util/azureclient/resources"
	"github.com/jim-minter/rp/pkg/util/azureclient/storage"
)

type Installer struct {
	log *logrus.Entry
	db  database.OpenShiftClusters

	domain string

	roleassignments        authorization.RoleAssignmentsClient
	disks                  compute.DisksClient
	virtualmachines        compute.VirtualMachinesClient
	recordsets             dns.RecordSetsClient
	userassignedidentities msi.UserAssignedIdentitiesClient
	interfaces             network.InterfacesClient
	publicipaddresses      network.PublicIPAddressesClient
	deployments            resources.DeploymentsClient
	groups                 resources.GroupsClient
	accounts               storage.AccountsClient
}

func NewInstaller(log *logrus.Entry, db database.OpenShiftClusters, domain string, authorizer autorest.Authorizer, subscriptionID string) *Installer {
	d := &Installer{
		log: log,
		db:  db,

		domain: domain,

		roleassignments:        authorization.NewRoleAssignmentsClient(subscriptionID, authorizer),
		disks:                  compute.NewDisksClient(subscriptionID),
		virtualmachines:        compute.NewVirtualMachinesClient(subscriptionID),
		recordsets:             dns.NewRecordSetsClient(subscriptionID, authorizer),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(subscriptionID),
		interfaces:             network.NewInterfacesClient(subscriptionID, authorizer),
		publicipaddresses:      network.NewPublicIPAddressesClient(subscriptionID, authorizer),
		deployments:            resources.NewDeploymentsClient(subscriptionID, authorizer),
		groups:                 resources.NewGroupsClient(subscriptionID, authorizer),
		accounts:               storage.NewAccountsClient(subscriptionID, authorizer),
	}

	d.disks.Authorizer = authorizer
	d.virtualmachines.Authorizer = authorizer
	d.userassignedidentities.Authorizer = authorizer
	return d
}

func (i *Installer) Install(ctx context.Context, doc *api.OpenShiftClusterDocument, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds) error {
	for {
		i.log.Printf("starting phase %s", doc.OpenShiftCluster.Properties.Install.Phase)
		switch doc.OpenShiftCluster.Properties.Install.Phase {
		case api.InstallPhaseDeployStorage:
			err := i.installStorage(ctx, doc, installConfig, platformCreds)
			if err != nil {
				return err
			}

		case api.InstallPhaseDeployResources:
			err := i.installResources(ctx, doc)
			if err != nil {
				return err
			}

		case api.InstallPhaseRemoveBootstrap:
			err := i.removeBootstrap(ctx, doc)
			if err != nil {
				return err
			}

			_, err = i.db.Patch(doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				doc.OpenShiftCluster.Properties.Install = nil
				return nil
			})
			return err

		default:
			return fmt.Errorf("unrecognised phase %s", doc.OpenShiftCluster.Properties.Install.Phase)
		}

		var err error
		doc, err = i.db.Patch(doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.Install.Phase++
			return nil
		})
		if err != nil {
			return err
		}
	}
}

func (i *Installer) getBlobService(ctx context.Context, oc *api.OpenShiftCluster) (azstorage.BlobStorageClient, error) {
	keys, err := i.accounts.ListKeys(ctx, oc.Properties.ResourceGroup, "cluster"+oc.Properties.StorageSuffix, "")
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	storage, err := azstorage.NewClient("cluster"+oc.Properties.StorageSuffix, *(*keys.Keys)[0].Value, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	return storage.GetBlobService(), nil
}

func (i *Installer) getGraph(ctx context.Context, oc *api.OpenShiftCluster) (graph, error) {
	i.log.Print("retrieving graph")

	blobService, err := i.getBlobService(ctx, oc)
	if err != nil {
		return nil, err
	}

	aro := blobService.GetContainerReference("aro")
	cluster := aro.GetBlobReference("graph")
	rc, err := cluster.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var g graph
	err = json.NewDecoder(rc).Decode(&g)
	if err != nil {
		return nil, err
	}

	return g, nil
}
