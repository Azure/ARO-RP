package alertmanager

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

type AlertManager interface {
	FetchPrometheusAlerts(ctx context.Context, alertmanagerService string) ([]model.Alert, error)
}

type alertManager struct {
	log        *logrus.Entry
	restConfig *rest.Config
}

func NewAlertManager(c *rest.Config, log *logrus.Entry) AlertManager {
	return &alertManager{log, c}
}

func (a *alertManager) FetchPrometheusAlerts(ctx context.Context, alertmanagerService string) ([]model.Alert, error) {
	httpClient := a.createHTTPClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alertmanagerService, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var alerts []model.Alert
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (a *alertManager) createHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}
				return portforward.DialContext(ctx, a.log, a.restConfig, "openshift-monitoring", "alertmanager-main-0", port)
			},
			DisableKeepAlives: true,
		},
	}
}
