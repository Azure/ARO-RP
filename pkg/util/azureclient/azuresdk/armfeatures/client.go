package armfeatures

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type FeaturesClient interface {
	Get(ctx context.Context, resourceProviderNamespace string, featureName string, options *armfeatures.ClientGetOptions) (armfeatures.ClientGetResponse, error)
}

type featuresClient struct {
	*armfeatures.Client
}

var _ FeaturesClient = &featuresClient{}

// NewDefaultFeaturesClient creates a new FeaturesClient with default options
func NewDefaultFeaturesClient(environment *azureclient.AROEnvironment, subscriptionID string, credential azcore.TokenCredential) (FeaturesClient, error) {
	options := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: environment.Cloud,
		},
	}

	return NewFeaturesClient(subscriptionID, credential, options)
}

// NewFeaturesClient creates a new FeaturesClient
func NewFeaturesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (FeaturesClient, error) {
	client, err := armfeatures.NewClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &featuresClient{
		Client: client,
	}, nil
}
