package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"context"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

const (
	alertmanagerService = "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts"
)

type Alert struct {
	AlertName string `json:"alertname"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
	Severity  string `json:"severity"`
	Summary   string `jsodn:"summary"`
}

func (c *client) GetOpenShiftFiringAlerts(ctx context.Context) ([]Alert, error) {
	alerts, err := c.fetcher.alertManagerClient.FetchPrometheusAlerts(ctx, alertmanagerService)
	if err != nil {
		return nil, err
	}

	firingAlerts := []Alert{}

	for _, alert := range alerts {
		if !namespace.IsOpenShiftNamespace(string(alert.Labels["namespace"])) {
			continue
		}

		if alert.Status() == "firing" {
			firingAlert := Alert{
				AlertName: alert.Name(),
				Status:    string(alert.Status()),
				Namespace: string(alert.Labels["namespace"]),
				Severity:  string(alert.Labels["severity"]),
				Summary:   string(alert.Annotations["summary"]),
			}
			firingAlerts = append(firingAlerts, firingAlert)
		}
	}
	return firingAlerts, nil
}
