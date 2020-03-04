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

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/prometheus/common/model"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

const (
	// alertNamespace is the namespace where the alert manager pod is living
	alertNamespace string = "openshift-monitoring"
	// alertPod is the pod to query
	alertPod string = "alertmanager-main-0"
	// alertServiceEndpoint is the service name to query
	alertServiceEndpoint string = "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts"
)

func (mon *Monitor) emitPrometheusAlerts(ctx context.Context) error {
	r, err := azure.ParseResourceID(mon.oc.ID)
	if err != nil {
		return err
	}

	hc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}

				// TODO: try other pods if -0 isn't available?
				return portforward.DialContext(ctx, mon.env, mon.oc, alertNamespace, alertPod, port)
			},
		},
	}

	resp, err := hc.Get(alertServiceEndpoint)
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
		// If the alert is still happening we are emitting
		if inTimeSpan(alert.StartsAt, alert.EndsAt, time.Now()) {
			alertmap[string(alert.Labels["alertname"])]++
		}
	}

	for alert, count := range alertmap {
		mon.m.EmitGauge("prometheus.alerts", count, map[string]string{
			"resourceId":     mon.oc.ID,
			"subscriptionId": r.SubscriptionID,
			"resourceGroup":  r.ResourceGroup,
			"resourceName":   r.ResourceName,
			"alert":          alert,
		})
	}

	return nil
}

func inTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}
