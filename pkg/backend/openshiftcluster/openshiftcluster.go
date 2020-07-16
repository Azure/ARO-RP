package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	pkgacrtoken "github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type ManagerInterface interface {
	Create(ctx context.Context) error
	Update(ctx context.Context) error
	AdminUpdate(ctx context.Context) error
	Delete(ctx context.Context) error
}

var _ ManagerInterface = &Manager{}

type Manager struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	billing      billing.Manager
	fpAuthorizer autorest.Authorizer

	ocDynamicValidator validate.OpenShiftClusterDynamicValidator

	groups         features.ResourceGroupsClient
	securityGroups network.SecurityGroupsClient

	dns             dns.Manager
	keyvault        keyvault.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager
	acrtoken        pkgacrtoken.Manager

	doc *api.OpenShiftClusterDocument
}

func NewManager(log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument) (ManagerInterface, error) {
	localFPAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	localFPKVAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := _env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	var acrtoken pkgacrtoken.Manager
	if _, ok := _env.(env.Dev); !ok {
		acrtoken, err = pkgacrtoken.NewManager(_env, localFPAuthorizer)
		if err != nil {
			return nil, err
		}
	}

	ocDynamicValidator, err := validate.NewOpenShiftClusterDynamicValidator(log, _env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:          log,
		env:          _env,
		db:           db,
		billing:      billing,
		fpAuthorizer: fpAuthorizer,

		ocDynamicValidator: ocDynamicValidator,

		groups:         features.NewResourceGroupsClient(subscriptionDoc.ID, fpAuthorizer),
		securityGroups: network.NewSecurityGroupsClient(subscriptionDoc.ID, fpAuthorizer),

		dns:             dns.NewManager(_env, localFPAuthorizer),
		keyvault:        keyvault.NewManager(localFPKVAuthorizer),
		privateendpoint: privateendpoint.NewManager(_env, localFPAuthorizer),
		acrtoken:        acrtoken,
		subnet:          subnet.NewManager(subscriptionDoc.ID, fpAuthorizer),

		doc: doc,
	}

	return m, nil
}
