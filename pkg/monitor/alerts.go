package monitor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/common/model"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

func (mon *monitor) validateAlerts(ctx context.Context, oc *api.OpenShiftCluster) error {
	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}

				return portforward.DialContext(ctx, mon.env, oc, AlertNamespace, AlertPodPrefix+"-0", port)
			},
		},
	}

	// TODO: try other pods if -0 isn't available?
	req, err := http.NewRequest(http.MethodGet, AlertServiceEndpoint, nil)
	if err != nil {
		return err
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var alerts []model.Alert
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		// If the alert is still happening we are emitting
		if inTimeSpan(alert.StartsAt, alert.EndsAt, time.Now()) {
			mon.clusterm.EmitGauge(MetricPrometheusAlert, 1, map[string]string{
				"resource": oc.ID,
				"alert":    string(alert.Labels["alertname"]),
			})
		}

	}

	return nil
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}
