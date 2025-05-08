package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"time"

	prometheusAPI "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// MetricValue contains the actual data of the metrics at certain timestamp, and a slice of this is used in the `Metrics` struct to combine all the metrics in one object.
type MetricValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// This is a structure which actually carries, a particular type of metrics
type Metrics struct {
	Name  string        `json:"metricname"`
	Value []MetricValue `json:"metricvalue"`
}

func (c *client) Statistics(ctx context.Context, httpClient *http.Client, promQuery string, duration time.Duration, endTime time.Time, prometheusURL string) ([]Metrics, error) {
	return c.fetcher.statistics(ctx, httpClient, promQuery, duration, endTime, prometheusURL)
}

func (f *realFetcher) statistics(ctx context.Context, httpClient *http.Client, promQuery string, duration time.Duration, endTime time.Time, prometheusURL string) ([]Metrics, error) {
	promConfig := prometheusAPI.Config{
		Address:      prometheusURL,
		RoundTripper: httpClient.Transport,
	}

	client, err := prometheusAPI.NewClient(promConfig)
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)
	startTime := endTime.Add(-1 * duration)
	value, warning, err := v1api.QueryRange(ctx, promQuery, v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  time.Minute * 2,
	})
	if len(warning) > 0 {
		f.log.Warn(warning)
	}
	if err != nil {
		return nil, err
	}

	valueMatrix := value.(model.Matrix)
	return convertToTypeMetrics(valueMatrix), nil
}

func convertToTypeMetrics(v model.Matrix) []Metrics {
	metrics := make([]Metrics, 0)
	for _, i := range v {
		metric := Metrics{}
		metric.Name = i.Metric.String()
		metricValues := make([]MetricValue, 0)
		for _, j := range i.Values {
			metricValues = append(metricValues, MetricValue{
				Timestamp: j.Timestamp.Time().Local().UTC(),
				Value:     float64(j.Value),
			})
		}
		metric.Value = metricValues
		metrics = append(metrics, metric)
	}

	return metrics
}

func GetPromQuery(statisticsType string) (string, error) {
	promQueries := map[string]string{
		"kubeapicodes":  "sum(rate(apiserver_request_total{job=\"apiserver\",code=~\"[45]..\"}[10m])) by (code, verb)",
		"kubeapicpu":    "rate(process_cpu_seconds_total{job=\"apiserver\"}[5m])",
		"kubeapimemory": "process_resident_memory_bytes{job=\"apiserver\"}",
		// kube-controller-manager
		"kubecontrollermanagercodes":  "sum(rate(rest_client_requests_total{job=\"kube-controller-manager\"}[5m])) by (code)",
		"kubecontrollermanagercpu":    "rate(process_cpu_seconds_total{job=\"kube-controller-manager\"}[5m])",
		"kubecontrollermanagermemory": "process_resident_memory_bytes{job=\"kube-controller-manager\"}",
		// DNS
		"dnsresponsecodes":    "sum(rate(coredns_dns_responses_total[5m])) by (rcode)",
		"dnserrorrate":        "sum(rate(coredns_dns_responses_total{rcode=~\"SERVFAIL|NXDOMAIN\"}[5m])) by (pod) / sum(rate(coredns_dns_responses_total{rcode=~\"NOERROR\"}[5m])) by (pod)",
		"dnshealthcheck":      "histogram_quantile(0.99, sum(rate(coredns_health_request_duration_seconds_bucket[5m])) by (le))",
		"dnsforwardedtraffic": "histogram_quantile(0.95, sum(rate(coredns_forward_request_duration_seconds_bucket[5m])) by (le))",
		"dnsalltraffic":       "histogram_quantile(0.95, sum(rate(coredns_dns_request_duration_seconds_bucket[5m])) by (le))",
		// Ingress
		"ingresscontrollercondition": "sum(ingress_controller_conditions) by (condition)",
	}
	promQuery, ok := promQueries[statisticsType]
	if !ok {
		return "", errors.New("invalid statistic type '" + statisticsType + "'")
	}
	return promQuery, nil
}
