package azmetrics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworksClient
type MetricsClient interface {
	QueryResources(ctx context.Context, subscriptionID string, metricNamespace string, metricNames []string, resourceIDs azmetrics.ResourceIDList, options *azmetrics.QueryResourcesOptions) (azmetrics.QueryResourcesResponse, error)
}

type metricsClient struct {
	*azmetrics.Client
}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewMetricsClient(region string, credential azcore.TokenCredential, armOptions *arm.ClientOptions) (MetricsClient, error) {
	var options *azmetrics.ClientOptions
	if armOptions != nil {
		options = &azmetrics.ClientOptions{
			ClientOptions: armOptions.ClientOptions,
		}
	}
	log.Infof("NewMetricsClient for %s with options %v => %v", region, armOptions, options)

	svc := options.Cloud.Services["query/azmetrics"]
	endpoint, err := url.Parse(svc.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics endpoint %s: %w", svc.Endpoint, err)
	}
	endpoint.Host = fmt.Sprintf("%s.%s", region, endpoint.Host)
	log.Infof("newMetricsClient: using endpoint %s", endpoint.String())
	client, err := azmetrics.NewClient(endpoint.String(), credential, options)

	return &metricsClient{client}, err
}
