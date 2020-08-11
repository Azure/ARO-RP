package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/openshift/installer/pkg/asset/bootstraplogging"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/sirupsen/logrus"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
	"github.com/Azure/ARO-RP/pkg/util/steps"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Installer contains information needed to install an ARO cluster
type Installer struct {
	log             *logrus.Entry
	env             env.Interface
	db              database.OpenShiftClusters
	billing         billing.Manager
	doc             *api.OpenShiftClusterDocument
	subscriptionDoc *api.SubscriptionDocument
	cipher          encryption.Cipher
	fpAuthorizer    refreshable.Authorizer

	disks             compute.DisksClient
	virtualmachines   compute.VirtualMachinesClient
	vnet              network.VirtualNetworksClient
	interfaces        network.InterfacesClient
	publicipaddresses network.PublicIPAddressesClient
	loadbalancers     network.LoadBalancersClient
	securitygroups    network.SecurityGroupsClient
	deployments       features.DeploymentsClient
	groups            features.ResourceGroupsClient
	accounts          storage.AccountsClient

	dns             dns.Manager
	keyvault        keyvault.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager

	kubernetescli kubernetes.Interface
	extcli        extensionsclient.Interface
	operatorcli   operatorclient.Interface
	configcli     configclient.Interface
	samplescli    samplesclient.Interface
	securitycli   securityclient.Interface
	arocli        aroclient.AroV1alpha1Interface
}

const deploymentName = "azuredeploy"

// NewInstaller creates a new Installer
func NewInstaller(ctx context.Context, log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters,
	billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument) (*Installer, error) {
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
		log:             log,
		env:             _env,
		db:              db,
		billing:         billing,
		cipher:          cipher,
		doc:             doc,
		subscriptionDoc: subscriptionDoc,
		fpAuthorizer:    fpAuthorizer,

		disks:             compute.NewDisksClient(r.SubscriptionID, fpAuthorizer),
		virtualmachines:   compute.NewVirtualMachinesClient(r.SubscriptionID, fpAuthorizer),
		vnet:              network.NewVirtualNetworksClient(r.SubscriptionID, fpAuthorizer),
		interfaces:        network.NewInterfacesClient(r.SubscriptionID, fpAuthorizer),
		publicipaddresses: network.NewPublicIPAddressesClient(r.SubscriptionID, fpAuthorizer),
		loadbalancers:     network.NewLoadBalancersClient(r.SubscriptionID, fpAuthorizer),
		securitygroups:    network.NewSecurityGroupsClient(r.SubscriptionID, fpAuthorizer),
		deployments:       features.NewDeploymentsClient(r.SubscriptionID, fpAuthorizer),
		groups:            features.NewResourceGroupsClient(r.SubscriptionID, fpAuthorizer),
		accounts:          storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),

		dns:             dns.NewManager(_env, localFPAuthorizer),
		keyvault:        keyvault.NewManager(localFPKVAuthorizer),
		privateendpoint: privateendpoint.NewManager(_env, localFPAuthorizer),
		subnet:          subnet.NewManager(r.SubscriptionID, fpAuthorizer),
	}, nil
}

// Update updates an ARO cluster
func (i *Installer) Update(ctx context.Context) error {
	steps := []steps.Step{
		steps.Action(i.initializeKubernetesClients),
		steps.Action(i.updateAzureCloudProvider),
		steps.Action(i.updateRoleAssignments),
	}

	return i.runSteps(ctx, steps)
}

// AdminUpgrade performs an admin upgrade of an ARO cluster
func (i *Installer) AdminUpgrade(ctx context.Context) error {
	steps := []steps.Step{
		steps.Action(i.initializeKubernetesClients), // must be first
		steps.Action(i.deploySnapshotUpgradeTemplate),
		steps.Action(i.startVMs),
		steps.Condition(i.apiServersReady, 30*time.Minute),
		steps.Action(i.ensureBillingRecord), // belt and braces
		steps.Action(i.fixLBProbes),
		steps.Action(i.fixNSG),
		steps.Action(i.ensureIfReload),
		steps.Action(i.ensureRouteFix),
		steps.Action(i.ensureAROOperator),
		steps.Condition(i.aroDeploymentReady, 10*time.Minute),
		steps.Action(i.upgradeCertificates),
		steps.Action(i.configureAPIServerCertificate),
		steps.Action(i.configureIngressCertificate),
		steps.Action(i.preUpgradeChecks), // Run this before Upgrade cluster
		steps.Action(i.upgradeCluster),
		steps.Action(i.addResourceProviderVersion), // Run this last so we capture the resource provider only once the upgrade has been fully performed
	}

	return i.runSteps(ctx, steps)
}

// Install installs an ARO cluster
func (i *Installer) Install(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image, bootstrapLoggingConfig *bootstraplogging.Config) error {
	steps := map[api.InstallPhase][]steps.Step{
		api.InstallPhaseBootstrap: {
			steps.Action(i.createDNS),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(func(ctx context.Context) error {
				return i.deployStorageTemplate(ctx, installConfig, platformCreds, image, bootstrapLoggingConfig)
			})),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(i.attachNSGsAndPatch)),
			steps.Action(i.ensureBillingRecord),
			steps.AuthorizationRefreshingAction(i.fpAuthorizer, steps.Action(i.deployResourceTemplate)),
			steps.Action(i.deployResourceTemplate),
			steps.Action(i.createPrivateEndpoint),
			steps.Action(i.updateAPIIP),
			steps.Action(i.createCertificates),
			steps.Action(i.initializeKubernetesClients),
			steps.Condition(i.bootstrapConfigMapReady, 30*time.Minute),
			steps.Action(i.ensureIfReload),
			steps.Action(i.ensureRouteFix),
			steps.Action(i.ensureAROOperator),
			steps.Action(i.incrInstallPhase),
		},
		api.InstallPhaseRemoveBootstrap: {
			steps.Action(i.initializeKubernetesClients),
			steps.Action(i.removeBootstrap),
			steps.Action(i.removeBootstrapIgnition),
			steps.Action(i.configureAPIServerCertificate),
			steps.Condition(i.apiServersReady, 30*time.Minute),
			steps.Condition(i.operatorConsoleExists, 30*time.Minute),
			steps.Action(i.updateConsoleBranding),
			steps.Condition(i.operatorConsoleReady, 10*time.Minute),
			steps.Condition(i.clusterVersionReady, 30*time.Minute),
			steps.Condition(i.aroDeploymentReady, 10*time.Minute),
			steps.Action(i.disableUpdates),
			steps.Action(i.disableSamples),
			steps.Action(i.disableOperatorHubSources),
			steps.Action(i.updateRouterIP),
			steps.Action(i.configureIngressCertificate),
			steps.Condition(i.ingressControllerReady, 30*time.Minute),
			steps.Action(i.finishInstallation),
			steps.Action(i.addResourceProviderVersion),
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
	return i.runSteps(ctx, steps[i.doc.OpenShiftCluster.Properties.Install.Phase])
}

func (i *Installer) runSteps(ctx context.Context, s []steps.Step) error {
	err := steps.Run(ctx, i.log, 10*time.Second, s)
	if err != nil {
		i.gatherFailureLogs(ctx)
	}
	return err
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

	i.extcli, err = extensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.operatorcli, err = operatorclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.securitycli, err = securityclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.samplescli, err = samplesclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.arocli, err = aroclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	i.configcli, err = configclient.NewForConfig(restConfig)
	return err
}

func (i *Installer) deployARMTemplate(ctx context.Context, rg string, tName string, t *arm.Template, params map[string]interface{}) error {
	i.log.Printf("deploying %s template", tName)

	err := i.deployments.CreateOrUpdateAndWait(ctx, rg, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   t,
			Parameters: params,
			Mode:       mgmtfeatures.Incremental,
		},
	})

	if azureerrors.IsDeploymentActiveError(err) {
		i.log.Printf("waiting for %s template to be deployed", tName)
		err = i.deployments.Wait(ctx, rg, deploymentName)
	}

	if azureerrors.HasAuthorizationFailedError(err) ||
		azureerrors.HasLinkedAuthorizationFailedError(err) {
		return err
	}

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

// addResourceProviderVersion sets the deploying resource provider version in
// the cluster document for deployment-tracking purposes.
func (i *Installer) addResourceProviderVersion(ctx context.Context) error {
	var err error
	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.ProvisionedBy = version.GitCommit
		return nil
	})
	return err
}
