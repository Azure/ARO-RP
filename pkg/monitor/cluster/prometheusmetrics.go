package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/portforward"
)

type prometheusMetrics struct {
	mon                *Monitor
	client             *http.Client
	prometheusQueryURL string
}

type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []prometheusResult `json:"result"`
	} `json:"data"`
}

type prometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

const prometheusQueryURL = "https://prometheus-k8s.openshift-monitoring.svc:9091/api/v1/query?query=%s"

func (mon *Monitor) emitPrometheusMetrics(ctx context.Context) error {
	mon.log.Debugf("running emitPrometheusMetrics")
	pm := &prometheusMetrics{
		mon:                mon,
		prometheusQueryURL: prometheusQueryURL,
	}

	err := pm.connectToPrometheus(ctx)
	if err != nil {
		return err
	}

	return pm.emitCNVMetrics(ctx)
}

func (pm *prometheusMetrics) connectToPrometheus(ctx context.Context) error {
	var err error

	for i := range 2 {
		pm.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					_, port, err := net.SplitHostPort(address)
					if err != nil {
						return nil, err
					}

					return pm.dialPrometheus(ctx, i, port)
				},
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				DisableKeepAlives: true,
			},
		}

		_, err = pm.queryPrometheus(ctx, fmt.Sprintf("prometheus_build_info{pod='prometheus-k8s-%d'}", i))
		if err == nil {
			pm.mon.log.Debugf("successfully queried prometheus-k8s-%d", i)
			return nil
		}
		pm.mon.log.Debugf("failed to query prometheus-k8s-%d: %+v", i, err)
	}

	return fmt.Errorf("failed to connect to any prometheus pod: %w", err)
}

func (pm *prometheusMetrics) dialPrometheus(ctx context.Context, podIndex int, port string) (net.Conn, error) {
	podName := fmt.Sprintf("prometheus-k8s-%d", podIndex)
	return portforward.DialContext(ctx, pm.mon.log, pm.mon.restconfig, "openshift-monitoring", podName, port)
}

func (pm *prometheusMetrics) queryPrometheus(ctx context.Context, query string) ([]prometheusResult, error) {
	queryURL := fmt.Sprintf(pm.prometheusQueryURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}

	token := pm.mon.restconfig.BearerToken
	if token == "" && pm.mon.restconfig.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(pm.mon.restconfig.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("load bearer token file: %w", err)
		}
		token = string(tokenBytes)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := pm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body on unexpected status code %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("unexpected status code %d %s: %s", resp.StatusCode, resp.Status, string(respBody))
	}

	var queryResp prometheusQueryResponse
	err = json.NewDecoder(resp.Body).Decode(&queryResp)
	if err != nil {
		return nil, err
	}

	if queryResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status %s: %+v", queryResp.Status, queryResp.Data)
	}

	return queryResp.Data.Result, nil
}

func (pm *prometheusMetrics) emitCNVMetrics(ctx context.Context) error {
	pm.mon.log.Debugf("emitting CNV metrics")
	results, err := pm.queryPrometheus(ctx, `{__name__="kubevirt_vmi_info"}`)
	if err != nil {
		return err
	}

	for _, result := range results {
		pm.mon.log.Debugf("emitting metric cnv.kubevirt.vmi.info with labels %+v", result.Metric)
		pm.mon.emitGauge("cnv.kubevirt.vmi.info", 1, result.Metric)
	}

	return nil
}
