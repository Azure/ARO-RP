package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/common/model"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

var ignoredAlerts = map[string]struct{}{
	"ImagePruningDisabled": {},
	"InsightsDisabled":     {},
}

type targetedAlertKey struct {
	alertName       string
	target          string
	secondaryTarget string
}

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

					return portforward.DialContext(ctx, mon.log, mon.restconfig, "openshift-monitoring", fmt.Sprintf("alertmanager-main-%d", i), port)
				},
				// HACK: without this, keepalive connections don't get closed,
				// resulting in excessive open TCP connections, lots of
				// goroutines not exiting and memory not being freed.
				// TODO: consider persisting hc between calls to Monitor().  If
				// this is done, take care in the future to call
				// hc.CloseIdleConnections() when finally disposing of an hc.
				DisableKeepAlives: true,
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

	mon.emitGauge("prometheus.alerts.count", int64(len(alerts)), nil)

	mon.aggregateAndEmitAlerts(alerts)

	return nil
}

// Given a slice of model.Alert this func aggregates them in two metrics:
// prometheus.alerts is used by many Geneva Monitors (example: MHCUnterminatedShortCircuit)
// prometheus.targeted.alerts will be used as replacement for some Monitor metrics whose metrics already exist in Prometheus
func (mon *Monitor) aggregateAndEmitAlerts(alerts []model.Alert) {
	collectedAlerts := map[string]struct {
		count    int64
		severity string
	}{}

	targetedAlerts := map[targetedAlertKey]struct {
		count    int64
		severity string
	}{}

	for _, alert := range alerts {
		if !namespace.IsOpenShiftNamespace(string(alert.Labels["namespace"])) {
			continue
		}

		if alertIsIgnored(alert.Name()) {
			continue
		}

		// Targeted alerts are decomposed by targets. In other words: an alert+target+secondary_target combination is unique
		// This allows us to do aggregations in Geneva, rather than having a single counter hiding all the label values.
		// Example: unhealthy nodes by condition, any unique combination of condition - node_name is unique, and it will have a dedicated statsd counter
		if isTargetedAlert(alert) {
			alertKey := targetedAlertKey{
				alertName:       alert.Name(),
				target:          string(alert.Labels["target"]),
				secondaryTarget: string(alert.Labels["secondary_target"]),
			}
			ta := targetedAlerts[alertKey]
			ta.severity = string(alert.Labels["severity"])
			ta.count++
			targetedAlerts[alertKey] = ta
			continue
		}

		a := collectedAlerts[alert.Name()]

		a.severity = string(alert.Labels["severity"])
		a.count++

		collectedAlerts[alert.Name()] = a
	}

	for alertName, a := range collectedAlerts {
		mon.emitGauge("prometheus.alerts", a.count, map[string]string{
			"alert":    alertName,
			"severity": a.severity,
		})
	}

	for alertKey, a := range targetedAlerts {
		mon.emitGauge("prometheus.targeted.alerts", a.count, map[string]string{
			"alert":            alertKey.alertName,
			"severity":         a.severity,
			"target":           alertKey.target,
			"secondary_target": alertKey.secondaryTarget,
		})
	}
}

func alertIsIgnored(alertName string) bool {
	// Customers using deprecated/removed APIs is not useful for us to scrape
	if strings.HasPrefix(alertName, "UsingDeprecatedAPI") {
		return true
	}
	if strings.HasPrefix(alertName, "APIRemovedInNext") {
		return true
	}

	if _, ok := ignoredAlerts[alertName]; ok {
		return true
	}

	return false
}

// Checks if the alert contains the labels "target" and "secondary_target" with values
func isTargetedAlert(alert model.Alert) bool {
	return alert.Labels["target"] != "" && alert.Labels["secondary_target"] != ""
}
