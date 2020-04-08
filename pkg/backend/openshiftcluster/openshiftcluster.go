package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	pkgacrtoken "github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Manager struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	billing      database.Billing
	fpAuthorizer autorest.Authorizer

	ocDynamicValidator validate.OpenShiftClusterDynamicValidator

	groups features.ResourceGroupsClient

	dns             dns.Manager
	keyvault        keyvault.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager
	acrtoken        acrtoken.Manager

	doc *api.OpenShiftClusterDocument
}

func NewManager(log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, billing database.Billing, doc *api.OpenShiftClusterDocument) (*Manager, error) {
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

	var acrtoken pkgacrtoken.Manager
	if _, ok := _env.(env.Dev); !ok {
		acrtoken, err = pkgacrtoken.NewManager(_env, localFPAuthorizer)
		if err != nil {
			return nil, err
		}
	}

	m := &Manager{
		log:          log,
		env:          _env,
		db:           db,
		billing:      billing,
		fpAuthorizer: fpAuthorizer,

		ocDynamicValidator: validate.NewOpenShiftClusterDynamicValidator(_env),

		groups: features.NewResourceGroupsClient(r.SubscriptionID, fpAuthorizer),

		dns:             dns.NewManager(_env, localFPAuthorizer),
		keyvault:        keyvault.NewManager(localFPKVAuthorizer),
		privateendpoint: privateendpoint.NewManager(_env, localFPAuthorizer),
		acrtoken:        acrtoken,
		subnet:          subnet.NewManager(r.SubscriptionID, fpAuthorizer),

		doc: doc,
	}

	return m, nil
}
