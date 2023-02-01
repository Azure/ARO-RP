package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/hive"
	"github.com/Azure/ARO-RP/pkg/metrics"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/deploy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/storage"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
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
	localFpAuthorizer refreshable.Authorizer
	metricsEmitter    metrics.Emitter

	spApplications        graphrbac.ApplicationsClient
	disks                 compute.DisksClient
	virtualMachines       compute.VirtualMachinesClient
	interfaces            network.InterfacesClient
	publicIPAddresses     network.PublicIPAddressesClient
	loadBalancers         network.LoadBalancersClient
	privateEndpoints      network.PrivateEndpointsClient
	securityGroups        network.SecurityGroupsClient
	deployments           features.DeploymentsClient
	resourceGroups        features.ResourceGroupsClient
	resources             features.ResourcesClient
	privateZones          privatedns.PrivateZonesClient
	virtualNetworkLinks   privatedns.VirtualNetworkLinksClient
	roleAssignments       authorization.RoleAssignmentsClient
	roleDefinitions       authorization.RoleDefinitionsClient
	denyAssignments       authorization.DenyAssignmentClient
	fpPrivateEndpoints    network.PrivateEndpointsClient
	rpPrivateLinkServices network.PrivateLinkServicesClient

	dns     dns.Manager
	storage storage.Manager
	subnet  subnet.Manager
	graph   graph.Manager

	kubernetescli    kubernetes.Interface
	extensionscli    extensionsclient.Interface
	maocli           machineclient.Interface
	mcocli           mcoclient.Interface
	operatorcli      operatorclient.Interface
	configcli        configclient.Interface
	samplescli       samplesclient.Interface
	securitycli      securityclient.Interface
	arocli           aroclient.Interface
	imageregistrycli imageregistryclient.Interface

	installViaHive     bool
	adoptViaHive       bool
	hiveClusterManager hive.ClusterManager

	aroOperatorDeployer deploy.Operator

	now func() time.Time

	openShiftClusterDocumentVersioner openShiftClusterDocumentVersioner
}

// New returns a cluster manager
func New(ctx context.Context, log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, dbGateway database.Gateway, dbOpenShiftVersions database.OpenShiftVersions, aead encryption.AEAD,
	billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument, hiveClusterManager hive.ClusterManager, metricsEmitter metrics.Emitter) (Interface, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := _env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	msiAuthorizer, err := _env.NewMSIAuthorizer(env.MSIContextRP, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	storage := storage.NewManager(_env, r.SubscriptionID, fpAuthorizer)

	installViaHive, err := _env.LiveConfig().InstallViaHive(ctx)
	if err != nil {
		return nil, err
	}

	adoptByHive, err := _env.LiveConfig().AdoptByHive(ctx)
	if err != nil {
		return nil, err
	}

	return &manager{
		log:                   log,
		env:                   _env,
		db:                    db,
		dbGateway:             dbGateway,
		dbOpenShiftVersions:   dbOpenShiftVersions,
		billing:               billing,
		doc:                   doc,
		subscriptionDoc:       subscriptionDoc,
		fpAuthorizer:          fpAuthorizer,
		localFpAuthorizer:     localFPAuthorizer,
		metricsEmitter:        metricsEmitter,
		disks:                 compute.NewDisksClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualMachines:       compute.NewVirtualMachinesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		interfaces:            network.NewInterfacesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		publicIPAddresses:     network.NewPublicIPAddressesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		loadBalancers:         network.NewLoadBalancersClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		privateEndpoints:      network.NewPrivateEndpointsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		securityGroups:        network.NewSecurityGroupsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		deployments:           features.NewDeploymentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		resourceGroups:        features.NewResourceGroupsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		resources:             features.NewResourcesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		privateZones:          privatedns.NewPrivateZonesClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualNetworkLinks:   privatedns.NewVirtualNetworkLinksClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		roleAssignments:       authorization.NewRoleAssignmentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		roleDefinitions:       authorization.NewRoleDefinitionsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		denyAssignments:       authorization.NewDenyAssignmentsClient(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		fpPrivateEndpoints:    network.NewPrivateEndpointsClient(_env.Environment(), _env.SubscriptionID(), localFPAuthorizer),
		rpPrivateLinkServices: network.NewPrivateLinkServicesClient(_env.Environment(), _env.SubscriptionID(), msiAuthorizer),

		dns:     dns.NewManager(_env, localFPAuthorizer),
		storage: storage,
		subnet:  subnet.NewManager(_env.Environment(), r.SubscriptionID, fpAuthorizer),
		graph:   graph.NewManager(log, aead, storage),

		installViaHive:                    installViaHive,
		adoptViaHive:                      adoptByHive,
		hiveClusterManager:                hiveClusterManager,
		now:                               func() time.Time { return time.Now() },
		openShiftClusterDocumentVersioner: new(openShiftClusterDocumentVersionerService),
	}, nil
}
