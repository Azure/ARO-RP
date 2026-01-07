package armquota

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/quota/armquota/v2"
)

type QuotaClient interface {
	Get(ctx context.Context, resourceName string, scope string, options *armquota.ClientGetOptions) (armquota.ClientGetResponse, error)
}

type quotaClient struct {
	*armquota.Client
}

var _ QuotaClient = &quotaClient{}

// NewQuotaClient creates a new client for azure quotas
func NewQuotaClient(credential azcore.TokenCredential, options *arm.ClientOptions) (QuotaClient, error) {
	clientFactory, err := armquota.NewClientFactory(credential, options)
	if err != nil {
		return nil, err
	}

	return &quotaClient{Client: clientFactory.NewClient()}, nil
}
