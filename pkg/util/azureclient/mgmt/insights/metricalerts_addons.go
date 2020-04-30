package insights

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
)

// MetricAlertsClientAddons contains addons for MetricAlertsClient
type MetricAlertsClientAddons interface {
	ListByResourceGroup(ctx context.Context, resourceGroupName string) (result *[]mgmtinsights.MetricAlertResource, err error)
	Delete(ctx context.Context, resourceGroupName string, ruleName string) (err error)
}

func (c *metricAlertsClient) ListByResourceGroup(ctx context.Context, resourceGroupName string) (result *[]mgmtinsights.MetricAlertResource, err error) {
	metricAlerts, err := c.MetricAlertsClient.ListByResourceGroup(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}
	return metricAlerts.Value, nil
}

func (c *metricAlertsClient) Delete(ctx context.Context, resourceGroupName string, ruleName string) (err error) {
	// Delete returns 200 or 204 no matter if the deleted alert rule existed, only the error matters
	_, err = c.MetricAlertsClient.Delete(ctx, resourceGroupName, ruleName)
	return err
}
