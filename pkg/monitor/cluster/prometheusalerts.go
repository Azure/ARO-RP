package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/common/model"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var ignoredAlerts = map[string]struct{}{
	"ImagePruningDisabled": {},
}

func (mon *Monitor) emitPrometheusAlerts(ctx context.Context) error {
	resp, err := mon.requestMetricHTTP(ctx, "alertmanager-main", "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts")
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
		if !namespace.IsOpenShift(string(alert.Labels["namespace"])) {
			continue
		}

		if alertIsIgnored(alert.Name()) {
			continue
		}

		a := m[string(alert.Name())]

		a.severity = string(alert.Labels["severity"])
		a.count++

		m[string(alert.Name())] = a
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
