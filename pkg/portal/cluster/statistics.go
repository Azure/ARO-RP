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

type MetricValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type Metrics struct {
	Name  string        `json:"metricname"`
	Value []MetricValue `json:"metricvalue"`
}

func (c *client) Statistics(ctx context.Context, httpClient *http.Client, promQuery string, duration string, endTime time.Time) ([]Metrics, error) {
	return c.fetcher.Statistics(ctx, httpClient, promQuery, duration, endTime)
}

func (f *realFetcher) Statistics(ctx context.Context, httpClient *http.Client, promQuery string, duration string, endTime time.Time) ([]Metrics, error) {
	promConfig := prometheusAPI.Config{
		Address: "http://prometheus-k8s-0:9090",
		Client:  httpClient,
	}

	client, err := prometheusAPI.NewClient(promConfig)
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)

	startTime, err := getStartTimeFromDuration(duration, endTime)
	if err != nil {
		return nil, err
	}
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

func getStartTimeFromDuration(duration string, endTime time.Time) (time.Time, error) {
	switch duration {
	case "1m":
		return endTime.Add(-1 * time.Minute), nil
	case "5m":
		return endTime.Add(-5 * time.Minute), nil
	case "10m":
		return endTime.Add(-10 * time.Minute), nil
	case "30m":
		return endTime.Add(-30 * time.Minute), nil
	case "1h":
		return endTime.Add(-1 * time.Hour), nil
	case "2h":
		return endTime.Add(-2 * time.Hour), nil
	case "6h":
		return endTime.Add(-6 * time.Hour), nil
	case "12h":
		return endTime.Add(-12 * time.Hour), nil
	case "1d":
		return endTime.Add(-24 * time.Hour), nil
	case "2d":
		return endTime.Add(-48 * time.Hour), nil
	case "1w":
		return endTime.Add(-7 * 24 * time.Hour), nil
	case "2w":
		return endTime.Add(-14 * 24 * time.Hour), nil
	case "4w":
		return endTime.Add(-28 * 24 * time.Hour), nil
	case "8w":
		return endTime.Add(-56 * 24 * time.Hour), nil
	}
	return time.Time{}, errors.New("invalid duration")
}

func GetPromQuery(statisticsType string) (string, error) {
	switch statisticsType {
	//kube-apiserver
	case "kubeapicodes":
		return "sum(rate(apiserver_request_total{job=\"apiserver\",code=~\"[45]..\"}[10m])) by (code, verb)", nil
	case "kubeapicpu":
		return "rate(process_cpu_seconds_total{job=\"apiserver\"}[5m])", nil
	case "kubeapimemory":
		return "process_resident_memory_bytes{job=\"apiserver\"}", nil
	//kube-controller-manager
	case "kubecontrollermanagercodes":
		return "sum(rate(rest_client_requests_total{job=\"kube-controller-manager\"}[5m])) by (code)", nil
	case "kubecontrollermanagercpu":
		return "rate(process_cpu_seconds_total{job=\"kube-controller-manager\"}[5m])", nil
	case "kubecontrollermanagermemory":
		return "process_resident_memory_bytes{job=\"kube-controller-manager\"}", nil
	//DNS
	case "dnsresponsecodes":
		return "sum(rate(coredns_dns_responses_total[5m])) by (rcode)", nil
	case "dnserrorrate":
		return "sum(rate(coredns_dns_responses_total{rcode=~\"SERVFAIL|NXDOMAIN\"}[5m])) by (pod) / sum(rate(coredns_dns_responses_total{rcode=~\"NOERROR\"}[5m])) by (pod)", nil
	case "dnshealthcheck":
		return "histogram_quantile(0.99, sum(rate(coredns_health_request_duration_seconds_bucket[5m])) by (le))", nil
	case "dnsforwardedtraffic":
		return "histogram_quantile(0.95, sum(rate(coredns_forward_request_duration_seconds_bucket[5m])) by (le))", nil
	case "dnsalltraffic":
		return "histogram_quantile(0.95, sum(rate(coredns_dns_request_duration_seconds_bucket[5m])) by (le))", nil
	//Ingress
	case "ingresscontrollercondition":
		return "sum(ingress_controller_conditions) by (condition)", nil

	default:
		return "", errors.New("invalid statistic type '" + statisticsType + "'")
	}
}
