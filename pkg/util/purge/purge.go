package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

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

	subnetManager subnet.Manager

	shouldDelete checkFn
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, subscriptionID string, shouldDelete checkFn, dryRun bool) (*ResourceCleaner, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &ResourceCleaner{
		log:    log,
		dryRun: dryRun,

		resourcegroupscli:      features.NewResourceGroupsClient(subscriptionID, authorizer),
		vnetscli:               network.NewVirtualNetworksClient(subscriptionID, authorizer),
		privatelinkservicescli: network.NewPrivateLinkServicesClient(subscriptionID, authorizer),
		securitygroupscli:      network.NewSecurityGroupsClient(subscriptionID, authorizer),

		subnetManager: subnet.NewManager(subscriptionID, authorizer),

		// ShouldDelete decides whether the resource group gets deleted
		shouldDelete: shouldDelete,
	}, nil
}
