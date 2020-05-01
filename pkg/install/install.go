package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/adminactions"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// Installer contains information needed to install an ARO cluster
type Installer struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	adminactions adminactions.Interface
	billing      billing.Manager
	doc          *api.OpenShiftClusterDocument
	cipher       encryption.Cipher
	fpAuthorizer autorest.Authorizer

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	loadbalancers     network.LoadBalancersClient
	deployments       features.DeploymentsClient
	groups            features.ResourceGroupsClient
	accounts          storage.AccountsClient

	dns             dns.Manager
	keyvault        keyvault.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager

	kubernetescli kubernetes.Interface
}

const (
	deploymentName = "azuredeploy"
	pollInterval   = 10 * time.Second
)

type action func(context.Context) error
type condition struct {
	f       wait.ConditionFunc
	timeout time.Duration
}

// NewInstaller creates a new Installer
func NewInstaller(ctx context.Context, log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, billing billing.Manager, doc *api.OpenShiftClusterDocument) (*Installer, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	localFPKVAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := _env.FPAuthorizer(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	return &Installer{
		log:          log,
		env:          _env,
		db:           db,
		billing:      billing,
		cipher:       cipher,
		doc:          doc,
		fpAuthorizer: fpAuthorizer,

		disks:             compute.NewDisksClient(r.SubscriptionID, fpAuthorizer),
		virtualmachines:   compute.NewVirtualMachinesClient(r.SubscriptionID, fpAuthorizer),
		interfaces:        network.NewInterfacesClient(r.SubscriptionID, fpAuthorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(r.SubscriptionID, fpAuthorizer),
		loadbalancers:     network.NewLoadBalancersClient(r.SubscriptionID, fpAuthorizer),
		deployments:       features.NewDeploymentsClient(r.SubscriptionID, fpAuthorizer),
		groups:            features.NewResourceGroupsClient(r.SubscriptionID, fpAuthorizer),
		accounts:          storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),

		dns:             dns.NewManager(_env, localFPAuthorizer),
		keyvault:        keyvault.NewManager(localFPKVAuthorizer),
		privateendpoint: privateendpoint.NewManager(_env, localFPAuthorizer),
		subnet:          subnet.NewManager(r.SubscriptionID, fpAuthorizer),
	}, nil
}

// Install installs an ARO cluster
func (i *Installer) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	steps := map[api.InstallPhase][]interface{}{
		api.InstallPhaseBootstrap: {
			action(i.createDNS),
			action(func(ctx context.Context) error {
				return i.deployStorageTemplate(ctx, installConfig, platformCreds, image)
			}),
			action(i.createBillingRecord),
			action(i.deployResourceTemplate),
			action(i.createPrivateEndpoint),
			action(i.updateAPIIP),
			action(i.createCertificates),
			action(i.initializeKubernetesClients),
			condition{i.adminactions.BootstrapConfigMapReady, 30 * time.Minute},
			action(i.adminactions.EnsureGenevaLogging),
			action(i.incrInstallPhase),
		},
		api.InstallPhaseRemoveBootstrap: {
			action(i.initializeKubernetesClients),
			action(i.removeBootstrap),
			action(i.removeBootstrapIgnition),
			action(i.configureAPIServerCertificate),
			condition{i.adminactions.APIServersReady, 30 * time.Minute},
			condition{i.adminactions.OperatorConsoleExists, 30 * time.Minute},
			action(i.adminactions.UpdateConsoleBranding),
			condition{i.adminactions.OperatorConsoleReady, 10 * time.Minute},
			condition{i.adminactions.ClusterVersionReady, 30 * time.Minute},
			action(i.adminactions.DisableAlertManagerWarning),
			action(i.adminactions.DisableUpdates),
			action(i.adminactions.DisableSamples),
			action(i.adminactions.DisableOperatorHubSources),
			action(i.updateRouterIP),
			action(i.configureIngressCertificate),
			condition{i.adminactions.IngressControllerReady, 30 * time.Minute},
			action(i.finishInstallation),
		},
	}

	err := i.startInstallation(ctx)
	if err != nil {
		return err
	}

	if steps[i.doc.OpenShiftCluster.Properties.Install.Phase] == nil {
		return fmt.Errorf("unrecognised phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	}
	i.log.Printf("starting phase %s", i.doc.OpenShiftCluster.Properties.Install.Phase)
	for _, step := range steps[i.doc.OpenShiftCluster.Properties.Install.Phase] {
		switch step := step.(type) {
		case action:
			i.log.Printf("running step %s", runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name())
			err = step(ctx)
			if err != nil {
				i.adminactions.GatherFailureLogs(ctx)
				return fmt.Errorf("%s: %s", runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name(), err)
			}
		case condition:
			i.log.Printf("waiting for %s", runtime.FuncForPC(reflect.ValueOf(step.f).Pointer()).Name())
			func() {
				timeoutCtx, cancel := context.WithTimeout(ctx, step.timeout)
				defer cancel()
				err = wait.PollImmediateUntil(pollInterval, step.f, timeoutCtx.Done())
			}()
			if err != nil {
				i.adminactions.GatherFailureLogs(ctx)
				return fmt.Errorf("%s: %s", runtime.FuncForPC(reflect.ValueOf(step.f).Pointer()).Name(), err)
			}
		default:
			return errors.New("install step must be an action or a condition")
		}
	}
	return nil
}

func (i *Installer) startInstallation(ctx context.Context) error {
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

func (i *Installer) finishInstallation(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.Install = nil
		return nil
	})
	return err
}

func (i *Installer) getBlobService(ctx context.Context, p mgmtstorage.Permissions, r mgmtstorage.SignedResourceTypes) (*azstorage.BlobStorageClient, error) {
	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	t := time.Now().UTC().Truncate(time.Second)
	res, err := i.accounts.ListAccountSAS(ctx, resourceGroup, "cluster"+i.doc.OpenShiftCluster.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               "b",
		ResourceTypes:          r,
		Permissions:            p,
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

func (i *Installer) graphExists(ctx context.Context) (bool, error) {
	i.log.Print("checking if graph exists")

	blobService, err := i.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	aro := blobService.GetContainerReference("aro")
	return aro.GetBlobReference("graph").Exists()
}

func (i *Installer) loadGraph(ctx context.Context) (graph, error) {
	i.log.Print("load graph")

	blobService, err := i.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
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

	blobService, err := i.getBlobService(ctx, mgmtstorage.Permissions("cw"), mgmtstorage.SignedResourceTypesO)
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

// initializeKubernetesClients initializes clients which are used
// once the cluster is up later on in the install process.
func (i *Installer) initializeKubernetesClients(ctx context.Context) error {
	restConfig, err := restconfig.RestConfig(i.env, i.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	i.kubernetescli, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.adminactions = adminactions.New(i.log, i.env, i.doc.OpenShiftCluster)
	return i.adminactions.InitializeClients(ctx)
}

func (i *Installer) deployARMTemplate(ctx context.Context, rg string, tName string, t *arm.Template, params map[string]interface{}) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		i.log.Printf("deploying %s template", tName)

		err := i.deployments.CreateOrUpdateAndWait(ctx, rg, deploymentName, mgmtfeatures.Deployment{
			Properties: &mgmtfeatures.DeploymentProperties{
				Template:   t,
				Parameters: params,
				Mode:       mgmtfeatures.Incremental,
			},
		})

		if isDeploymentActiveError(err) {
			i.log.Printf("waiting for %s template to be deployed", tName)
			err = i.deployments.Wait(ctx, rg, deploymentName)
		}

		if hasAuthorizationFailedError(err) {
			i.log.Print(err)

			// https://github.com/Azure/ARO-RP/issues/541: it is unclear if
			// this refresh helps or not
			if development, ok := i.env.(env.Dev); ok {
				err = development.RefreshFPAuthorizer(ctx, i.fpAuthorizer)
				if err != nil {
					return false, err
				}
			}

			return false, nil
		}

		return err == nil, err
	}, timeoutCtx.Done())

	serviceErr, _ := err.(*azure.ServiceError) // futures return *azure.ServiceError directly

	// CreateOrUpdate() returns a wrapped *azure.ServiceError
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		serviceErr, _ = detailedErr.Original.(*azure.ServiceError)
	}

	if serviceErr != nil {
		b, _ := json.Marshal(serviceErr)

		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: string(b),
					},
				},
			},
		}
	}

	return err
}
