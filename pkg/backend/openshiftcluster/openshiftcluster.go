package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/resources"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type Manager struct {
	log          *logrus.Entry
	env          env.Interface
	db           database.OpenShiftClusters
	fpAuthorizer autorest.Authorizer

	groups resources.GroupsClient

	subnets subnet.Manager

	doc *api.OpenShiftClusterDocument
}

func NewManager(log *logrus.Entry, env env.Interface, db database.OpenShiftClusters, doc *api.OpenShiftClusterDocument) (*Manager, error) {
	r, err := azure.ParseResourceID(doc.OpenShiftCluster.ID)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := env.FPAuthorizer(doc.OpenShiftCluster.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		log:          log,
		env:          env,
		db:           db,
		fpAuthorizer: fpAuthorizer,

		subnets: subnet.NewManager(r.SubscriptionID, fpAuthorizer),
		groups:  resources.NewGroupsClient(r.SubscriptionID, fpAuthorizer),

		doc: doc,
	}

	return m, nil
}
