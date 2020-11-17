package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	pkgacrtoken "github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/billing"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/privateendpoint"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Manager interface {
	Create(ctx context.Context) error
	Update(ctx context.Context) error
	AdminUpdate(ctx context.Context) error
	Delete(ctx context.Context) error
}

var _ Manager = &manager{}

type manager struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	cipher       encryption.Cipher
	billing      billing.Manager
	fpAuthorizer autorest.Authorizer

	ocDynamicValidator validate.OpenShiftClusterDynamicValidator

	resourceGroups features.ResourceGroupsClient
	securityGroups network.SecurityGroupsClient

	dns             dns.Manager
	privateendpoint privateendpoint.Manager
	subnet          subnet.Manager
	acrtoken        pkgacrtoken.Manager

	doc             *api.OpenShiftClusterDocument
	subscriptionDoc *api.SubscriptionDocument
}

// NewManager returns a new openshiftcluster Manager
func NewManager(log *logrus.Entry, _env env.Interface, db database.OpenShiftClusters, cipher encryption.Cipher, billing billing.Manager, doc *api.OpenShiftClusterDocument, subscriptionDoc *api.SubscriptionDocument) (Manager, error) {
	localFPAuthorizer, err := _env.FPAuthorizer(_env.TenantID(), _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := _env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, _env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	var acrtoken pkgacrtoken.Manager
	if _env.DeploymentMode() != deployment.Development {
		acrtoken, err = pkgacrtoken.NewManager(_env, localFPAuthorizer)
		if err != nil {
			return nil, err
		}
	}

	ocDynamicValidator, err := validate.NewOpenShiftClusterDynamicValidator(log, _env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	m := &manager{
		log:          log,
		env:          _env,
		db:           db,
		cipher:       cipher,
		billing:      billing,
		fpAuthorizer: fpAuthorizer,

		ocDynamicValidator: ocDynamicValidator,

		resourceGroups: features.NewResourceGroupsClient(subscriptionDoc.ID, fpAuthorizer),
		securityGroups: network.NewSecurityGroupsClient(subscriptionDoc.ID, fpAuthorizer),

		dns:             dns.NewManager(_env, localFPAuthorizer),
		privateendpoint: privateendpoint.NewManager(_env, localFPAuthorizer),
		acrtoken:        acrtoken,
		subnet:          subnet.NewManager(subscriptionDoc.ID, fpAuthorizer),

		doc:             doc,
		subscriptionDoc: subscriptionDoc,
	}

	return m, nil
}
