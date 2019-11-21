package install

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
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
	resources              resources.Client
	accounts               storage.AccountsClient
}

func NewInstaller(log *logrus.Entry, db database.OpenShiftClusters, domain string, authorizer autorest.Authorizer, subscriptionID string) *Installer {
	d := &Installer{
		log: log,
		db:  db,

		roleassignments:        authorization.NewRoleAssignmentsClient(subscriptionID),
		disks:                  compute.NewDisksClient(subscriptionID),
		virtualmachines:        compute.NewVirtualMachinesClient(subscriptionID),
		recordsets:             dns.NewRecordSetsClient(subscriptionID),
		userassignedidentities: msi.NewUserAssignedIdentitiesClient(subscriptionID),
		interfaces:             network.NewInterfacesClient(subscriptionID),
		publicipaddresses:      network.NewPublicIPAddressesClient(subscriptionID),
		deployments:            resources.NewDeploymentsClient(subscriptionID),
		groups:                 resources.NewGroupsClient(subscriptionID),
		resources:              resources.NewClient(subscriptionID),
		accounts:               storage.NewAccountsClient(subscriptionID),
	}

	d.roleassignments.Authorizer = authorizer
	d.disks.Authorizer = authorizer
	d.virtualmachines.Authorizer = authorizer
	d.recordsets.Authorizer = authorizer
	d.userassignedidentities.Authorizer = authorizer
	d.interfaces.Authorizer = authorizer
	d.publicipaddresses.Authorizer = authorizer
	d.deployments.Authorizer = authorizer
	d.groups.Authorizer = authorizer
	d.resources.Authorizer = authorizer
	d.accounts.Authorizer = authorizer

	d.deployments.Client.PollingDuration = time.Hour

	return d
}

func (i *Installer) Install(ctx context.Context, doc *api.OpenShiftClusterDocument, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds) error {
	for {
		i.log.Printf("starting phase %s", doc.OpenShiftCluster.Properties.Installation.Phase)
		switch doc.OpenShiftCluster.Properties.Installation.Phase {
		case api.InstallationPhaseDeployStorage:
			err := i.installStorage(ctx, doc, installConfig, platformCreds)
			if err != nil {
				return err
			}

		case api.InstallationPhaseDeployResources:
			err := i.installResources(ctx, doc)
			if err != nil {
				return err
			}

		case api.InstallationPhaseRemoveBootstrap:
			err := i.removeBootstrap(ctx, doc)
			if err != nil {
				return err
			}

			_, err = i.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
				doc.OpenShiftCluster.Properties.Installation = nil
				return nil
			})
			return err

		default:
			return fmt.Errorf("unrecognised phase %s", doc.OpenShiftCluster.Properties.Installation.Phase)
		}

		var err error
		doc, err = i.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.Installation.Phase++
			return nil
		})
		if err != nil {
			return err
		}
	}
}

func (i *Installer) getBlobService(ctx context.Context, doc *api.OpenShiftClusterDocument) (azstorage.BlobStorageClient, error) {
	keys, err := i.accounts.ListKeys(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "cluster"+doc.OpenShiftCluster.Properties.StorageSuffix, "")
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	storage, err := azstorage.NewClient("cluster"+doc.OpenShiftCluster.Properties.StorageSuffix, *(*keys.Keys)[0].Value, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	return storage.GetBlobService(), nil
}

func (i *Installer) getGraph(ctx context.Context, doc *api.OpenShiftClusterDocument) (graph, error) {
	i.log.Print("retrieving graph")

	blobService, err := i.getBlobService(ctx, doc)
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
