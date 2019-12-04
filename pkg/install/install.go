package install

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/util/azureclient/compute"
	"github.com/jim-minter/rp/pkg/util/azureclient/network"
	"github.com/jim-minter/rp/pkg/util/azureclient/resources"
	"github.com/jim-minter/rp/pkg/util/azureclient/storage"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

type Installer struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	fpAuthorizer autorest.Authorizer

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	deployments       resources.DeploymentsClient
	groups            resources.GroupsClient
	accounts          storage.AccountsClient

	subnets subnet.Manager
}

func NewInstaller(log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, fpAuthorizer autorest.Authorizer, subscriptionID string) *Installer {
	return &Installer{
		log:          log,
		env:          env,
		db:           db,
		fpAuthorizer: fpAuthorizer,

		disks:             compute.NewDisksClient(subscriptionID, fpAuthorizer),
		virtualmachines:   compute.NewVirtualMachinesClient(subscriptionID, fpAuthorizer),
		interfaces:        network.NewInterfacesClient(subscriptionID, fpAuthorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(subscriptionID, fpAuthorizer),
		deployments:       resources.NewDeploymentsClient(subscriptionID, fpAuthorizer),
		groups:            resources.NewGroupsClient(subscriptionID, fpAuthorizer),
		accounts:          storage.NewAccountsClient(subscriptionID, fpAuthorizer),

		subnets: subnet.NewManager(subscriptionID, fpAuthorizer),
	}
}

func (i *Installer) Install(ctx context.Context, doc *api.OpenShiftClusterDocument, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds) error {
	doc, err := i.db.Patch(doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			doc.OpenShiftCluster.Properties.Install = &api.Install{}
		}
		return nil
	})
	if err != nil {
		return err
	}

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

func (i *Installer) getBlobService(ctx context.Context, oc *api.OpenShiftCluster) (*azstorage.BlobStorageClient, error) {
	t := time.Now().UTC().Truncate(time.Second)

	res, err := i.accounts.ListAccountSAS(ctx, oc.Properties.ResourceGroup, "cluster"+oc.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               "b",
		ResourceTypes:          "o",
		Permissions:            "crw",
		Protocols:              mgmtstorage.HTTPS,
		SharedAccessStartTime:  &date.Time{Time: t},
		SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
	})
	if err != nil {
		return nil, err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return nil, err
	}

	c := azstorage.NewAccountSASClient("cluster"+oc.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	return &c, nil
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
