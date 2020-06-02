package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	"context"
	"os"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type checkFn func(mgmtfeatures.ResourceGroup, *logrus.Entry) bool

// ResourceCleaner hold the context required for cleaning
type ResourceCleaner struct {
	appMap map[string][]string

	log    *logrus.Entry
	dryRun bool

	resourcegroupscli      features.ResourceGroupsClient
	vnetscli               network.VirtualNetworksClient
	privatelinkservicescli network.PrivateLinkServicesClient
	securitygroupscli      network.SecurityGroupsClient

	applicationscli     graphrbac.ApplicationsClient
	roleassignmentcli   authorization.RoleAssignmentsClient
	serviceprincipalcli graphrbac.ServicePrincipalClient

	subnetManager subnet.Manager

	shouldDelete checkFn
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, shouldDelete checkFn, dryRun bool) (*ResourceCleaner, error) {
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	rbAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	rc := &ResourceCleaner{
		appMap: map[string][]string{},
		log:    log,
		dryRun: dryRun,

		resourcegroupscli:      features.NewResourceGroupsClient(subscriptionID, authorizer),
		vnetscli:               network.NewVirtualNetworksClient(subscriptionID, authorizer),
		privatelinkservicescli: network.NewPrivateLinkServicesClient(subscriptionID, authorizer),
		securitygroupscli:      network.NewSecurityGroupsClient(subscriptionID, authorizer),

		applicationscli:   graphrbac.NewApplicationsClient(tenantID, rbAuthorizer),
		roleassignmentcli: authorization.NewRoleAssignmentsClient(subscriptionID, authorizer),

		subnetManager: subnet.NewManager(subscriptionID, authorizer),

		// ShouldDelete decides whether the resource group gets deleted
		shouldDelete: shouldDelete,
	}

	err = rc.prepareApps(context.Background())
	if err != nil {
		return nil, err
	}

	return rc, nil
}
