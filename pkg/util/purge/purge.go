package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/common"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
)

type checkFn func(mgmtfeatures.ResourceGroup, *logrus.Entry) bool

// ResourceCleaner hold the context required for cleaning
type ResourceCleaner struct {
	log    *logrus.Entry
	dryRun bool

	resourcegroupscli      features.ResourceGroupsClient
	privatelinkservicescli armnetwork.PrivateLinkServicesClient
	securitygroupscli      armnetwork.SecurityGroupsClient

	subnet armnetwork.SubnetsClient

	shouldDelete checkFn
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, env env.Core, shouldDelete checkFn, dryRun bool) (*ResourceCleaner, error) {
	options := env.Environment().EnvironmentCredentialOptions()
	spTokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		return nil, err
	}

	scopes := []string{env.Environment().ResourceManagerScope}
	authorizer := azidext.NewTokenCredentialAdapter(spTokenCredential, scopes)

	clientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud:           env.Environment().Cloud,
			Retry:           common.RetryOptions,
			PerCallPolicies: []policy.Policy{azureclient.NewLoggingPolicy()},
		},
	}

	privateLinkServiceClient, err := armnetwork.NewPrivateLinkServicesClient(env.SubscriptionID(), spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	securityGroupsClient, err := armnetwork.NewSecurityGroupsClient(env.SubscriptionID(), spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	subnetGroupsClient, err := armnetwork.NewSubnetsClient(env.SubscriptionID(), spTokenCredential, clientOptions)
	if err != nil {
		return nil, err
	}

	return &ResourceCleaner{
		log:    log,
		dryRun: dryRun,

		resourcegroupscli:      features.NewResourceGroupsClient(env.Environment(), env.SubscriptionID(), authorizer),
		privatelinkservicescli: privateLinkServiceClient,
		securitygroupscli:      securityGroupsClient,
		subnet:                 subnetGroupsClient,

		// ShouldDelete decides whether the resource group gets deleted
		shouldDelete: shouldDelete,
	}, nil
}
