package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
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
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// Installer contains information needed to install an ARO cluster
type Installer struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	doc          *api.OpenShiftClusterDocument
	cipher       encryption.Cipher
	fpAuthorizer autorest.Authorizer

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	loadbalancers     network.LoadBalancersClient
	deployments       resources.DeploymentsClient
	groups            resources.GroupsClient
	accounts          storage.AccountsClient

	dns             dns.Manager
	keyvault        keyvault.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager
}

// NewInstaller creates a new Installer
func NewInstaller(ctx context.Context, log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, doc *api.OpenShiftClusterDocument) (*Installer, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	localFPKVAuthorizer, err := env.FPAuthorizer(env.TenantID(), azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, env)
	if err != nil {
		return nil, err
	}

	return &Installer{
		log:          log,
		env:          env,
		db:           db,
		cipher:       cipher,
		doc:          doc,
		fpAuthorizer: fpAuthorizer,

		disks:             compute.NewDisksClient(r.SubscriptionID, fpAuthorizer),
		virtualmachines:   compute.NewVirtualMachinesClient(r.SubscriptionID, fpAuthorizer),
		interfaces:        network.NewInterfacesClient(r.SubscriptionID, fpAuthorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(r.SubscriptionID, fpAuthorizer),
		loadbalancers:     network.NewLoadBalancersClient(r.SubscriptionID, fpAuthorizer),
		deployments:       resources.NewDeploymentsClient(r.SubscriptionID, fpAuthorizer),
		groups:            resources.NewGroupsClient(r.SubscriptionID, fpAuthorizer),
		accounts:          storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),

		dns:             dns.NewManager(env, localFPAuthorizer),
		keyvault:        keyvault.NewManager(env, localFPKVAuthorizer),
		privateendpoint: privateendpoint.NewManager(env, localFPAuthorizer),
		subnet:          subnet.NewManager(r.SubscriptionID, fpAuthorizer),
	}, nil
}

// Install installs an ARO cluster
func (i *Installer) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	steps := map[api.InstallPhase][]func(context.Context) error{
		api.InstallPhaseDeployStorage: {
			i.createDNS,
			func(ctx context.Context) error {
				return i.installStorage(ctx, installConfig, platformCreds, image)
			},
			i.incrInstallPhase,
		},
		api.InstallPhaseDeployResources: {
			i.installResources,
			i.createPrivateEndpoint,
			i.updateAPIIP,
			i.waitForBootstrapConfigmap,
			i.incrInstallPhase,
		},
		api.InstallPhaseRemoveBootstrap: {
			i.removeBootstrap,
			i.updateConsoleBranding,
			i.waitForClusterVersion,
			i.disableUpdates,
			i.updateRouterIP,
			i.endOfInstallPhase,
		},
	}

	err := i.startInstallPhase(ctx)
	if err != nil {
		return err
	}

	if steps[i.doc.OpenShiftCluster.Properties.Install.Phase] == nil {
		return fmt.Errorf("unrecognised phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	}
	i.log.Printf("starting phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	for _, step := range steps[i.doc.OpenShiftCluster.Properties.Install.Phase] {
		i.log.Printf("running step %s", runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name())
		err := step(ctx)
		if i.doc.OpenShiftCluster.Properties.Install == nil && err == nil {
			// sucessful end of the install process
			break
		}
		if err != nil {
			break
		}
	}

	return err
}

func (i *Installer) startInstallPhase(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		if doc.OpenShiftCluster.Properties.Install == nil {
			doc.OpenShiftCluster.Properties.Install = &api.Install{}
		}
		return nil
	})
	return err
}

func (i *Installer) incrInstallPhase(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install.Phase++
		return nil
	})
	return err
}

func (i *Installer) endOfInstallPhase(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install = nil
		return nil
	})
	return err
}

func (i *Installer) getBlobService(ctx context.Context) (*azstorage.BlobStorageClient, error) {
	resourceGroup := i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID[strings.LastIndexByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')+1:]

	t := time.Now().UTC().Truncate(time.Second)

	res, err := i.accounts.ListAccountSAS(ctx, resourceGroup, "cluster"+i.doc.OpenShiftCluster.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
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

func (i *Installer) loadGraph(ctx context.Context) (graph, error) {
	i.log.Print("load graph")

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

	encrypted, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	output, err := i.cipher.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	var g graph
	err = json.Unmarshal(output, &g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (i *Installer) saveGraph(ctx context.Context, g graph) error {
	i.log.Print("save graph")

	blobService, err := i.getBlobService(ctx)
	if err != nil {
		return err
	}

	bootstrap := g[reflect.TypeOf(&bootstrap.Bootstrap{})].(*bootstrap.Bootstrap)
	bootstrapIgn := blobService.GetContainerReference("ignition").GetBlobReference("bootstrap.ign")
	err = bootstrapIgn.CreateBlockBlobFromReader(bytes.NewReader(bootstrap.File.Data), nil)
	if err != nil {
		return err
	}

	graph := blobService.GetContainerReference("aro").GetBlobReference("graph")
	b, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		return err
	}

	output, err := i.cipher.Encrypt(b)
	if err != nil {
		return err
	}

	return graph.CreateBlockBlobFromReader(bytes.NewReader([]byte(output)), nil)
}
