package cluster

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

	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

func (mon *Monitor) emitPrometheusAlerts(ctx context.Context) error {
	var resp *http.Response
	var err error

	for i := 0; i < 3; i++ {
		hc := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					_, port, err := net.SplitHostPort(address)
					if err != nil {
						return nil, err
					}

					return portforward.DialContext(ctx, mon.env, mon.oc, "openshift-monitoring", fmt.Sprintf("alertmanager-main-%d", i), port)
				},
			},
		}

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts", nil)
		if err != nil {
			return err
		}

		resp, err = hc.Do(req)
		if err == nil {
			break
		}
	}
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

	alertmap := map[string]int64{}

	for _, alert := range alerts {
		if alert.Name() == "UsingDeprecatedAPIExtensionsV1Beta1" {
			continue
		}

		// If the alert we are emitting is still happening
		if inTimeSpan(alert.StartsAt, alert.EndsAt, time.Now()) {
			alertmap[string(alert.Labels["alertname"])]++
		}
	}

	for alert, count := range alertmap {
		mon.emitGauge("prometheus.alerts", count, map[string]string{
			"alert": alert,
		})
	}

	return nil
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}
