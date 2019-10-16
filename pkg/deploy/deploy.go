package deploy

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
)

type Deployer struct {
	log *logrus.Entry
	db  database.OpenShiftClusters

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	recordsets        dns.RecordSetsClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	deployments       resources.DeploymentsClient
	groups            resources.GroupsClient
	resources         resources.Client
	accounts          storage.AccountsClient
}

func NewDeployer(log *logrus.Entry, db database.OpenShiftClusters, authorizer autorest.Authorizer, subscriptionID string) *Deployer {
	d := &Deployer{
		log: log,
		db:  db,

		disks:             compute.NewDisksClient(subscriptionID),
		virtualmachines:   compute.NewVirtualMachinesClient(subscriptionID),
		recordsets:        dns.NewRecordSetsClient(subscriptionID),
		interfaces:        network.NewInterfacesClient(subscriptionID),
		publicipaddresses: network.NewPublicIPAddressesClient(subscriptionID),
		deployments:       resources.NewDeploymentsClient(subscriptionID),
		groups:            resources.NewGroupsClient(subscriptionID),
		resources:         resources.NewClient(subscriptionID),
		accounts:          storage.NewAccountsClient(subscriptionID),
	}

	d.disks.Authorizer = authorizer
	d.virtualmachines.Authorizer = authorizer
	d.recordsets.Authorizer = authorizer
	d.interfaces.Authorizer = authorizer
	d.publicipaddresses.Authorizer = authorizer
	d.deployments.Authorizer = authorizer
	d.groups.Authorizer = authorizer
	d.resources.Authorizer = authorizer
	d.accounts.Authorizer = authorizer

	return d
}

func (d *Deployer) Deploy(ctx context.Context, doc *api.OpenShiftClusterDocument, installConfig *installconfig.InstallConfig) error {
	for {
		d.log.Printf("starting phase %s", doc.OpenShiftCluster.Properties.Installation.Phase)
		switch doc.OpenShiftCluster.Properties.Installation.Phase {
		case api.InstallationPhaseDeployStorage:
			err := d.deployStorage(ctx, doc, installConfig)
			if err != nil {
				return err
			}

		case api.InstallationPhaseDeployResources:
			err := d.deployResources(ctx, doc)
			if err != nil {
				return err
			}

		case api.InstallationPhaseRemoveBootstrap:
			err := d.removeBootstrap(ctx, doc)
			if err != nil {
				return err
			}

			_, err = d.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
				doc.OpenShiftCluster.Properties.Installation = nil
				return nil
			})
			return err

		default:
			return fmt.Errorf("unrecognised phase %s", doc.OpenShiftCluster.Properties.Installation.Phase)
		}

		var err error
		doc, err = d.db.Patch(doc.OpenShiftCluster.ID, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.Installation.Phase++
			return nil
		})
		if err != nil {
			return err
		}
	}
}

func (d *Deployer) getBlobService(ctx context.Context, doc *api.OpenShiftClusterDocument) (azstorage.BlobStorageClient, error) {
	keys, err := d.accounts.ListKeys(ctx, doc.OpenShiftCluster.Properties.ResourceGroup, "cluster"+doc.OpenShiftCluster.Properties.StorageSuffix)
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	storage, err := azstorage.NewClient("cluster"+doc.OpenShiftCluster.Properties.StorageSuffix, *(*keys.Keys)[0].Value, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return azstorage.BlobStorageClient{}, err
	}

	return storage.GetBlobService(), nil
}

func (d *Deployer) getGraph(ctx context.Context, doc *api.OpenShiftClusterDocument) (Graph, error) {
	d.log.Print("retrieving graph")

	blobService, err := d.getBlobService(ctx, doc)
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

	var g Graph
	err = json.NewDecoder(rc).Decode(&g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func randomLowerCaseAlphanumericString(n int) (string, error) {
	return randomString("abcdefghijklmnopqrstuvwxyz0123456789", n)
}

func randomString(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

func restConfig(adminClient *kubeconfig.AdminClient) (*rest.Config, error) {
	config, err := clientcmd.Load(adminClient.File.Data)
	if err != nil {
		return nil, err
	}

	return clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
}
