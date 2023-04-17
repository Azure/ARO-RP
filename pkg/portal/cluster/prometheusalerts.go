package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
import (
	"context"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

type FiringAlert struct {
	AlertName string `json:"alertname"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
	Severity  string `json:"severity"`
	Summary   string `jsodn:"summary"`
}

func (c *client) FiringAlerts(ctx context.Context) ([]FiringAlert, error) {
	alerts, err := c.fetcher.alertManagerClient.FetchPrometheusAlerts(ctx)
	if err != nil {
		return nil, err
	}

	firingAlerts := []FiringAlert{}

	for _, alert := range alerts {
		if !namespace.IsOpenShiftNamespace(string(alert.Labels["namespace"])) {
			continue
		}

		if alert.Status() == "firing" {
			firingAlert := FiringAlert{
				AlertName: alert.Name(),
				Status:    string(alert.Status()),
				Namespace: string(alert.Labels["namespace"]),
				Severity:  string(alert.Labels["severity"]),
				Summary:   string(alert.Annotations["summary"]),
			}
			firingAlerts = append(firingAlerts, firingAlert)
		}
	}
	fmt.Print(firingAlerts)
	return firingAlerts, nil
}
