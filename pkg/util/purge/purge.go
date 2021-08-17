package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type checkFn func(mgmtfeatures.ResourceGroup, *logrus.Entry) bool

// ResourceCleaner hold the context required for cleaning
type ResourceCleaner struct {
	log    *logrus.Entry
	dryRun bool

	resourcegroupscli      features.ResourceGroupsClient
	vnetscli               network.VirtualNetworksClient
	privatelinkservicescli network.PrivateLinkServicesClient
	securitygroupscli      network.SecurityGroupsClient

	subnet subnet.Manager

	shouldDelete checkFn
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, env env.Core, shouldDelete checkFn, dryRun bool) (*ResourceCleaner, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &ResourceCleaner{
		log:    log,
		dryRun: dryRun,

		resourcegroupscli:      features.NewResourceGroupsClient(env.Environment(), env.SubscriptionID(), authorizer),
		vnetscli:               network.NewVirtualNetworksClient(env.Environment(), env.SubscriptionID(), authorizer),
		privatelinkservicescli: network.NewPrivateLinkServicesClient(env.Environment(), env.SubscriptionID(), authorizer),
		securitygroupscli:      network.NewSecurityGroupsClient(env.Environment(), env.SubscriptionID(), authorizer),

		subnet: subnet.NewManager(env.Environment(), env.SubscriptionID(), authorizer),

		// ShouldDelete decides whether the resource group gets deleted
		shouldDelete: shouldDelete,
	}, nil
}
