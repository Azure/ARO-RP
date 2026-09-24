package armpolicyinsights

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/policyinsights/armpolicyinsights"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type PolicyRestrictionsClient interface {
	CheckAtResourceGroupScope(ctx context.Context, resourceGroupName string, parameters armpolicyinsights.CheckRestrictionsRequest, options *armpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeOptions) (armpolicyinsights.PolicyRestrictionsClientCheckAtResourceGroupScopeResponse, error)
}

type policyRestrictionsClient struct {
	*armpolicyinsights.PolicyRestrictionsClient
}

func NewPolicyRestrictionsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (PolicyRestrictionsClient, error) {
	clientFactory, err := armpolicyinsights.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &policyRestrictionsClient{clientFactory.NewPolicyRestrictionsClient()}, nil
}
