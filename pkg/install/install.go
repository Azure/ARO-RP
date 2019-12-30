package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

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
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Installer struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	doc          *api.OpenShiftClusterDocument
	fpAuthorizer autorest.Authorizer

	privateendpoint privateendpoint.Manager
	dns             dns.Manager

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	deployments       resources.DeploymentsClient
	groups            resources.GroupsClient
	accounts          storage.AccountsClient

	subnets subnet.Manager
}

func NewInstaller(log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, doc *api.OpenShiftClusterDocument) (*Installer, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return &Installer{
		log:          log,
		env:          env,
		db:           db,
		doc:          doc,
		fpAuthorizer: fpAuthorizer,

		privateendpoint: privateendpoint.NewManager(env, localFPAuthorizer),
		dns:             dns.NewManager(env, localFPAuthorizer),

		disks:             compute.NewDisksClient(r.SubscriptionID, fpAuthorizer),
		virtualmachines:   compute.NewVirtualMachinesClient(r.SubscriptionID, fpAuthorizer),
		interfaces:        network.NewInterfacesClient(r.SubscriptionID, fpAuthorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(r.SubscriptionID, fpAuthorizer),
		deployments:       resources.NewDeploymentsClient(r.SubscriptionID, fpAuthorizer),
		groups:            resources.NewGroupsClient(r.SubscriptionID, fpAuthorizer),
		accounts:          storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),

		subnets: subnet.NewManager(r.SubscriptionID, fpAuthorizer),
	}, nil
}

func (i *Installer) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	var err error

	i.doc, err = i.db.Patch(i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			doc.OpenShiftCluster.Properties.Install = &api.Install{}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for {
		i.log.Printf("starting phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
		switch i.doc.OpenShiftCluster.Properties.Install.Phase {
		case api.InstallPhaseDeployStorage:
			err := i.installStorage(ctx, installConfig, platformCreds, image)
			if err != nil {
				return err
			}

		case api.InstallPhaseDeployResources:
			err := i.installResources(ctx)
			if err != nil {
				return err
			}

		case api.InstallPhaseRemoveBootstrap:
			err := i.removeBootstrap(ctx)
			if err != nil {
				return err
			}

			i.doc, err = i.db.Patch(i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
				doc.OpenShiftCluster.Properties.Install = nil
				return nil
			})
			return err

		default:
			return fmt.Errorf("unrecognised phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
		}

		i.doc, err = i.db.Patch(i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.Install.Phase++
			return nil
		})
		if err != nil {
			return err
		}
	}
}

func (i *Installer) getBlobService(ctx context.Context) (*azstorage.BlobStorageClient, error) {
	t := time.Now().UTC().Truncate(time.Second)

	res, err := i.accounts.ListAccountSAS(ctx, i.doc.OpenShiftCluster.Properties.ResourceGroup, "cluster"+i.doc.OpenShiftCluster.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
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

	c := azstorage.NewAccountSASClient("cluster"+i.doc.OpenShiftCluster.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	return &c, nil
}

func (i *Installer) getGraph(ctx context.Context) (graph, error) {
	i.log.Print("retrieving graph")

	blobService, err := i.getBlobService(ctx)
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
