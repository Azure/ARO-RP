package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	samplesclient "github.com/openshift/client-go/samples/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
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
	log               *logrus.Entry
	env               env.Interface
	db                database.OpenShiftClusters
	billing           billing.Manager
	doc               *api.OpenShiftClusterDocument
	subscriptionDoc   *api.SubscriptionDocument
	cipher            encryption.Cipher
	fpAuthorizer      refreshable.Authorizer
	localFpAuthorizer refreshable.Authorizer

	disks               compute.DisksClient
	virtualMachines     compute.VirtualMachinesClient
	interfaces          network.InterfacesClient
	publicIPAddresses   network.PublicIPAddressesClient
	loadBalancers       network.LoadBalancersClient
	securityGroups      network.SecurityGroupsClient
	deployments         features.DeploymentsClient
	resourceGroups      features.ResourceGroupsClient
	resources           features.ResourcesClient
	virtualNetworkLinks privatedns.VirtualNetworkLinksClient
	storageAccounts     storage.AccountsClient

	dns             dns.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager

	kubernetescli kubernetes.Interface
	extensionscli extensionsclient.Interface
	operatorcli   operatorclient.Interface
	configcli     configclient.Interface
	samplescli    samplesclient.Interface
	securitycli   securityclient.Interface
	arocli        aroclient.AroV1alpha1Interface
}

const deploymentName = "azuredeploy"

// New returns a cluster manager
func New(ctx context.Context, log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, cipher encryption.Cipher,
	billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument) (Interface, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	localFPAuthorizer, err := env.FPAuthorizer(env.TenantID(), env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return &manager{
		log:               log,
		env:               env,
		db:                db,
		billing:           billing,
		doc:               doc,
		subscriptionDoc:   subscriptionDoc,
		cipher:            cipher,
		fpAuthorizer:      fpAuthorizer,
		localFpAuthorizer: localFPAuthorizer,

		disks:               compute.NewDisksClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualMachines:     compute.NewVirtualMachinesClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		interfaces:          network.NewInterfacesClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		publicIPAddresses:   network.NewPublicIPAddressesClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		loadBalancers:       network.NewLoadBalancersClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		securityGroups:      network.NewSecurityGroupsClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		deployments:         features.NewDeploymentsClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		resourceGroups:      features.NewResourceGroupsClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		resources:           features.NewResourcesClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		virtualNetworkLinks: privatedns.NewVirtualNetworkLinksClient(env.Environment(), r.SubscriptionID, fpAuthorizer),
		storageAccounts:     storage.NewAccountsClient(env.Environment(), r.SubscriptionID, fpAuthorizer),

		dns:             dns.NewManager(env, localFPAuthorizer),
		privateendpoint: privateendpoint.NewManager(env, localFPAuthorizer),
		subnet:          subnet.NewManager(env, r.SubscriptionID, fpAuthorizer),
	}, nil
}
