package features

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armfeatures"
)

// SubscriptionFeatureRegistrationsClient is a minimal interface for azure SubscriptionFeatureRegistrationsClient
type SubscriptionFeatureRegistrationsClient interface {
	Get(ctx context.Context, providerNamespace string, featureName string, options *armfeatures.SubscriptionFeatureRegistrationsClientGetOptions) (armfeatures.SubscriptionFeatureRegistrationsClientGetResponse, error)
}
