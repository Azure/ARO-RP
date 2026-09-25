package armresources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

type TagsClient interface {
	UpdateAtScope(ctx context.Context, scope string, parameters armresources.TagsPatchResource, options *armresources.TagsClientUpdateAtScopeOptions) (armresources.TagsClientUpdateAtScopeResponse, error)
}

type tagsClient struct {
	*armresources.TagsClient
}

func NewTagsClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (TagsClient, error) {
	clientFactory, err := armresources.NewClientFactory(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}
	return &tagsClient{clientFactory.NewTagsClient()}, nil
}
