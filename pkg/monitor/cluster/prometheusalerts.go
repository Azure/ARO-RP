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

	m := map[string]struct {
		count    int64
		severity string
	}{}

	mon.emitGauge("prometheus.alerts.count", int64(len(alerts)), nil)

	for _, alert := range alerts {
		if !namespace.IsOpenShiftNamespace(string(alert.Labels["namespace"])) {
			continue
		}

		if alertIsIgnored(alert.Name()) {
			continue
		}

		a := m[alert.Name()]

		a.severity = string(alert.Labels["severity"])
		a.count++

		m[alert.Name()] = a
	}

	for alertName, a := range m {
		mon.emitGauge("prometheus.alerts", a.count, map[string]string{
			"alert":    alertName,
			"severity": a.severity,
		})
	}

	return nil
}

func alertIsIgnored(alertName string) bool {
	if strings.HasPrefix(alertName, "UsingDeprecatedAPI") {
		return true
	}

	if _, ok := ignoredAlerts[alertName]; ok {
		return true
	}

	return false
}
