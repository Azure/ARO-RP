package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armauthorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armmsi"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/blob"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	utilgraph "github.com/Azure/ARO-RP/pkg/util/graph"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/storage"
	"github.com/Azure/ARO-RP/pkg/util/token"
)

type Interface interface {
	Install(ctx context.Context) error
	Delete(ctx context.Context) error
	Update(ctx context.Context) error
	AdminUpdate(ctx context.Context) error
}

// manager contains information needed to install and maintain an ARO cluster
type manager struct {
	log                 *logrus.Entry
	env                 env.Interface
	db                  database.OpenShiftClusters
	dbGateway           database.Gateway
	dbOpenShiftVersions database.OpenShiftVersions

	billing           billing.Manager
	doc               *api.OpenShiftClusterDocument
	subscriptionDoc   *api.SubscriptionDocument
	fpAuthorizer      refreshable.Authorizer
	localFpAuthorizer autorest.Authorizer
	metricsEmitter    metrics.Emitter

	spGraphClient                 *utilgraph.GraphServiceClient
	disks                         compute.DisksClient
	virtualMachines               compute.VirtualMachinesClient
	resourceSkus                  compute.ResourceSkusClient
	armInterfaces                 armnetwork.InterfacesClient
	armPublicIPAddresses          armnetwork.PublicIPAddressesClient
	armLoadBalancers              armnetwork.LoadBalancersClient
	armPrivateEndpoints           armnetwork.PrivateEndpointsClient
	armSecurityGroups             armnetwork.SecurityGroupsClient
	deployments                   features.DeploymentsClient
	resourceGroups                features.ResourceGroupsClient
	resources                     features.ResourcesClient
	privateZones                  privatedns.PrivateZonesClient
	virtualNetworkLinks           privatedns.VirtualNetworkLinksClient
	roleAssignments               authorization.RoleAssignmentsClient
	roleDefinitions               authorization.RoleDefinitionsClient
	armRoleDefinitions            armauthorization.RoleDefinitionsClient
	denyAssignments               authorization.DenyAssignmentClient
	armFPPrivateEndpoints         armnetwork.PrivateEndpointsClient
	armRPPrivateLinkServices      armnetwork.PrivateLinkServicesClient
	armClusterPrivateLinkServices armnetwork.PrivateLinkServicesClient
	armSubnets                    armnetwork.SubnetsClient
	userAssignedIdentities        armmsi.UserAssignedIdentitiesClient

	dns     dns.Manager
	storage storage.Manager
	graph   graph.Manager
	rpBlob  blob.Manager

	ch               clienthelper.Interface
	kubernetescli    kubernetes.Interface
	dynamiccli       dynamic.Interface
	extensionscli    extensionsclient.Interface
	maocli           machineclient.Interface
	mcocli           mcoclient.Interface
	operatorcli      operatorclient.Interface
	configcli        configclient.Interface
	samplescli       samplesclient.Interface
	securitycli      securityclient.Interface
	arocli           aroclient.Interface
	imageregistrycli imageregistryclient.Interface

	installViaHive       bool
	adoptViaHive         bool
	hiveClusterManager   hive.ClusterManager
	fpServicePrincipalID string

	aroOperatorDeployer deploy.Operator

	msiDataplane                           dataplane.ClientFactory
	clusterMsiKeyVaultStore                azsecrets.Client
	clusterMsiFederatedIdentityCredentials armmsi.FederatedIdentityCredentialsClient

	now func() time.Time

	openShiftClusterDocumentVersioner openShiftClusterDocumentVersioner

	platformWorkloadIdentityRolesByVersion platformworkloadidentity.PlatformWorkloadIdentityRolesByVersion
	platformWorkloadIdentities             map[string]api.PlatformWorkloadIdentity

	time time.Time
}

// New returns a cluster manager
func New(ctx context.Context, log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, dbGateway database.Gateway, dbOpenShiftVersions database.OpenShiftVersions, dbPlatformWorkloadIdentityRoleSets database.PlatformWorkloadIdentityRoleSets, aead encryption.AEAD,
	billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument, hiveClusterManager hive.ClusterManager, metricsEmitter metrics.Emitter,
) (Interface, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), nil, _env.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	// TODO: Delete once the replacement to track2 is done
	fpAuthorizer, err := refreshable.NewAuthorizer(_env, subscriptionDoc.Subscription.Properties.TenantID)
	if err != nil {
		return nil, err
	}

	fpCredClusterTenant, err := _env.FPNewClientCertificateCredential(subscriptionDoc.Subscription.Properties.TenantID, nil)
	if err != nil {
		return nil, err
	}

	t, err := fpCredClusterTenant.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{_env.Environment().ResourceManagerScope}})
	if err != nil {
		return nil, err
	}
	tokenClaims, err := token.ExtractClaims(t.Token)
	if err != nil {
		return nil, err
	}
	fpspID := tokenClaims.ObjectId

	fpCredRPTenant, err := _env.FPNewClientCertificateCredential(_env.TenantID(), nil)
	if err != nil {
		return nil, err
	}

	msiCredential, err := _env.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	installViaHive, err := _env.LiveConfig().InstallViaHive(ctx)
	if err != nil {
		return nil, err
	}

	adoptByHive, err := _env.LiveConfig().AdoptByHive(ctx)
	if err != nil {
		return nil, err
	}

	clientOptions := _env.Environment().ArmClientOptions()

	armInterfacesClient, err := armnetwork.NewInterfacesClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armPublicIPAddressesClient, err := armnetwork.NewPublicIPAddressesClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armLoadBalancersClient, err := armnetwork.NewLoadBalancersClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armPrivateEndpoints, err := armnetwork.NewPrivateEndpointsClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armFPPrivateEndpoints, err := armnetwork.NewPrivateEndpointsClient(_env.SubscriptionID(), fpCredRPTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armSecurityGroupsClient, err := armnetwork.NewSecurityGroupsClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armRPPrivateLinkServices, err := armnetwork.NewPrivateLinkServicesClient(_env.SubscriptionID(), msiCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	clusterRPPrivateLinkServices, err := armnetwork.NewPrivateLinkServicesClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	storage, err := storage.NewManager(r.SubscriptionID, _env.Environment().StorageEndpointSuffix, fpCredClusterTenant, doc.OpenShiftCluster.UsesWorkloadIdentity(), clientOptions)
	if err != nil {
		return nil, err
	}

	rpBlob, err := blob.NewManager(_env.SubscriptionID(), msiCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	armSubnetsClient, err := armnetwork.NewSubnetsClient(r.SubscriptionID, fpCredClusterTenant, clientOptions)
	if err != nil {
		return nil, err
	}

	armRoleDefinitionsClient, err := armauthorization.NewArmRoleDefinitionsClient(fpCredClusterTenant, r.SubscriptionID, clientOptions)
	if err != nil {
		return nil, err
	}

	platformWorkloadIdentityRolesByVersion := platformworkloadidentity.NewPlatformWorkloadIdentityRolesByVersionService()

	m := &manager{
		log:                           log,
		env:                           _env,
		db:                            db,
		dbGateway:                     dbGateway,
		dbOpenShiftVersions:           dbOpenShiftVersions,
		billing:                       billing,
		doc:                           doc,
		subscriptionDoc:               subscriptionDoc,
		fpAuthorizer:                  fpAuthorizer,
		localFpAuthorizer:             localFPAuthorizer,
		metricsEmitter:                metricsEmitter,
		disks:                         compute.NewDisksClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualMachines:               compute.NewVirtualMachinesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		resourceSkus:                  compute.NewResourceSkusClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		armInterfaces:                 armInterfacesClient,
		armPublicIPAddresses:          armPublicIPAddressesClient,
		armLoadBalancers:              armLoadBalancersClient,
		armPrivateEndpoints:           armPrivateEndpoints,
		armSecurityGroups:             armSecurityGroupsClient,
		deployments:                   features.NewDeploymentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		resourceGroups:                features.NewResourceGroupsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		resources:                     features.NewResourcesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		privateZones:                  privatedns.NewPrivateZonesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualNetworkLinks:           privatedns.NewVirtualNetworkLinksClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		roleAssignments:               authorization.NewRoleAssignmentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		roleDefinitions:               authorization.NewRoleDefinitionsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		armRoleDefinitions:            armRoleDefinitionsClient,
		denyAssignments:               authorization.NewDenyAssignmentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		armFPPrivateEndpoints:         armFPPrivateEndpoints,
		armRPPrivateLinkServices:      armRPPrivateLinkServices,
		armClusterPrivateLinkServices: clusterRPPrivateLinkServices,
		armSubnets:                    armSubnetsClient,

		dns:                                    dns.NewManager(_env, fpCredRPTenant),
		storage:                                storage,
		graph:                                  graph.NewManager(_env, log, aead, storage),
		rpBlob:                                 rpBlob,
		installViaHive:                         installViaHive,
		adoptViaHive:                           adoptByHive,
		hiveClusterManager:                     hiveClusterManager,
		now:                                    func() time.Time { return time.Now() },
		openShiftClusterDocumentVersioner:      new(openShiftClusterDocumentVersionerService),
		platformWorkloadIdentityRolesByVersion: platformWorkloadIdentityRolesByVersion,
		fpServicePrincipalID:                   fpspID,

		time: time.Now(),
	}

	if doc.OpenShiftCluster.UsesWorkloadIdentity() {
		if m.doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateDeleting {
			err = m.platformWorkloadIdentityRolesByVersion.PopulatePlatformWorkloadIdentityRolesByVersion(ctx, doc.OpenShiftCluster, dbPlatformWorkloadIdentityRoleSets)
			if err != nil {
				return nil, err
			}
		}

		msiResourceId, err := m.doc.OpenShiftCluster.ClusterMsiResourceId()
		if err != nil {
			return nil, err
		}

		var msiDataplane dataplane.ClientFactory
		if _env.FeatureIsSet(env.FeatureUseMockMsiRp) {
			msiDataplane = _env.MockMSIResponses(msiResourceId)
		} else {
			msiDataplaneClientOptions, err := _env.MsiDataplaneClientOptions(doc.CorrelationData)
			if err != nil {
				return nil, err
			}

			// MSI dataplane client receives tenant from the bearer challenge, so we can't limit the allowed tenants in the credential
			fpMSICred, err := _env.FPNewClientCertificateCredential(_env.TenantID(), []string{"*"})
			if err != nil {
				return nil, err
			}

			msiDataplane = dataplane.NewClientFactory(fpMSICred, _env.MsiRpEndpoint(), msiDataplaneClientOptions)
		}
		m.msiDataplane = msiDataplane

		secretsClient, err := azsecrets.NewClient(azsecrets.URI(_env, _env.ClusterMsiKeyVaultName(), ""), msiCredential, _env.Environment().AzureClientOptions())
		if err != nil {
			return nil, fmt.Errorf("cannot create MSI key vault client: %w", err)
		}
		m.clusterMsiKeyVaultStore = secretsClient
	}

	return m, nil
}

func (m *manager) APICertName() string {
	return m.doc.ID + "-apiserver"
}

func (m *manager) IngressCertName() string {
	return m.doc.ID + "-ingress"
}
