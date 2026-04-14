package armmonitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sdkarmmonitor "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
)

// MetricsClient is a client for querying Azure Monitor metrics via ARM.
type MetricsClient interface {
	List(ctx context.Context, resourceURI string, options *sdkarmmonitor.MetricsClientListOptions) (sdkarmmonitor.MetricsClientListResponse, error)
}

// NewMetricsClient creates a new MetricsClient that queries via the ARM
// control plane (management.azure.com). This avoids the separate OAuth2
// audience required by the data plane batch API.
func NewMetricsClient(subscriptionID string, credential azcore.TokenCredential, armOptions *arm.ClientOptions) (MetricsClient, error) {
	return sdkarmmonitor.NewMetricsClient(subscriptionID, credential, armOptions)
}
