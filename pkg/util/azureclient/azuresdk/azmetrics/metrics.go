package azmetrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// MetricsClient is a client for querying Azure Monitor metrics.
type MetricsClient interface {
	QueryResources(ctx context.Context, subscriptionID string, metricNamespace string, metricNames []string, resourceIDs azmetrics.ResourceIDList, options *azmetrics.QueryResourcesOptions) (azmetrics.QueryResourcesResponse, error)
}

type metricsClient struct {
	*azmetrics.Client
}

// NewMetricsClient creates a new MetricsClient.
func NewMetricsClient(region string, credential azcore.TokenCredential, armOptions *arm.ClientOptions) (MetricsClient, error) {
	options := &azmetrics.ClientOptions{}
	if armOptions != nil {
		options.ClientOptions = armOptions.ClientOptions
	}
	if len(options.Cloud.Services) == 0 {
		options.Cloud = cloud.AzurePublic
	}
	svc, ok := options.Cloud.Services[azmetrics.ServiceName]
	if !ok || svc.Audience == "" {
		return nil, fmt.Errorf("metrics audience for %q service not configured in cloud configuration", azmetrics.ServiceName)
	}
	endpoint, err := url.Parse(svc.Audience)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics audience %s: %w", svc.Audience, err)
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("metrics audience %q is not a valid URL", svc.Audience)
	}
	endpoint.Host = fmt.Sprintf("%s.%s", region, endpoint.Host)
	client, err := azmetrics.NewClient(endpoint.String(), credential, options)
	if err != nil {
		return nil, err
	}

	return &metricsClient{client}, nil
}
