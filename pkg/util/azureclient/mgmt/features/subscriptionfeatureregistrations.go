package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"
)

// SubscriptionFeatureRegistrationsClient is a minimal interface for azure SubscriptionFeatureRegistrationsClient
type SubscriptionFeatureRegistrationsClient interface {
	Get(ctx context.Context, providerNamespace string, featureName string, options *armfeatures.SubscriptionFeatureRegistrationsClientGetOptions) (armfeatures.SubscriptionFeatureRegistrationsClientGetResponse, error)
}

type subscriptionFeatureRegistrationsClient struct {
	armfeatures.SubscriptionFeatureRegistrationsClient
}

var _ SubscriptionFeatureRegistrationsClient = &subscriptionFeatureRegistrationsClient{}

// NewSubscriptionFeatureRegistrationsClient creates a new SubscriptionFeatureRegistrationsClient
func NewSubscriptionFeatureRegistrationsClient(subscriptionID string, credential *azidentity.ClientCertificateCredential, options *arm.ClientOptions) (SubscriptionFeatureRegistrationsClient, error) {
	client, err := armfeatures.NewSubscriptionFeatureRegistrationsClient(subscriptionID, credential, options)
	if err != nil {
		return nil, err
	}

	return &subscriptionFeatureRegistrationsClient{
		SubscriptionFeatureRegistrationsClient: *client,
	}, nil
}
