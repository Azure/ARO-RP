package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
)

const (
	ErrMsgMetricDataUnavailable     = "the required metric data is unavailable"
	ErrMsgNotEnoughMetricData       = "not enough metric data points"
	ErrMsgUnknownPrometheusResponse = "unknown prometheus response"
	ErrMsgInvalidPrometheusResponse = "invalid prometheus response"
	ErrMsgPrometheusQueryFailed     = "prometheus query failed"
	ErrMsgPrometheusQueryUnexpected = "prometheus query returned unexpected type or value"

	ProceedDownsizeMsg      = "assessment successful, proceed with downsize"
	DoNotProceedDownsizeMsg = "assessment unsuccessful, do not proceed with downsize"
	IncompleteDownsizeMsg   = "assessment incomplete due to errors"
)

var percentageAvailableQueries = [][]string{
	{
		"node_memory_MemTotal_bytes",
		// This query retrieves the percentage of available metric data points collected for
		// each master node and returns the minimum percentage. This is a conservative check
		// for ensuring that each node has enough metric data points for resize assessments.
		// The total number of metric data points can be calculated as;
		//   duration / prometheus scrape interval
		// Duration is for 2 weeks which is 14d * 24h * 60m * 60s. The default prometheus scrape
		// interval is 15s for node-exporter, see cluster's prometheus config for more information.
		"min(count_over_time(node_memory_MemTotal_bytes{instance=~\".*master.*\"}[2w])) / ((14 * 24 * 60 * 60) / 15) * 100",
	},
}

type controlPlaneResize struct {
	portForward        adminactions.PortForwardActions
	portForwardService adminactions.PortForwardService
}

type portForwardService struct {
	portForwardPodName   string
	portForwardNamespace string
	portForwardPort      string
}

type MetricData struct {
	Name             string  `json:"name"`
	PercentAvailable float64 `json:"percentAvailable"`
	Query            string  `json:"query"`
}

type DownsizeAssessment struct {
	Proceed         bool         `json:"proceed"`
	Recommendation  string       `json:"recommendation"`
	Err             string       `json:"error,omitempty"`
	MetricDataStats []MetricData `json:"metricDataStats,omitempty"`
}

type PrometheusData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}

type VectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  [2]interface{}    `json:"value"`
}

func newControlPlaneResize(portForward adminactions.PortForwardActions) *controlPlaneResize {
	portFwdSvc := portForwardService{
		portForwardPodName:   "prometheus-k8s-0",
		portForwardNamespace: "openshift-monitoring",
		portForwardPort:      "9090",
	}
	return &controlPlaneResize{
		portForward:        portForward,
		portForwardService: portFwdSvc,
	}
}

func (p *controlPlaneResize) createPrometheusQueryRequest(ctx context.Context, query string) (*http.Request, error) {
	promURL := fmt.Sprintf(
		"http://%s.%s.svc:%s/api/v1/query",
		p.portForwardService.GetPortForwardPodName(),
		p.portForwardService.GetPortForwardNamespace(),
		p.portForwardService.GetPortForwardPort(),
	)
	params := url.Values{}
	params.Add("query", query)
	fullURL := fmt.Sprintf("%s?%s", promURL, params.Encode())
	return http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
}

func (p *controlPlaneResize) parsePrometheusResponse(resp *http.Response) (*PrometheusResponse, error) {
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	var result PrometheusResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	return &result, nil
}

func (p *controlPlaneResize) parseVectorResult(resp *PrometheusResponse) ([]VectorResult, error) {
	var vector []VectorResult
	if resp.Data.ResultType == "vector" {
		json.Unmarshal(resp.Data.Result, &vector)
		return vector, nil
	}
	return nil, fmt.Errorf("result type %s is not a vector: %w", resp.Data.ResultType, ErrMsgInvalidPrometheusResponse)
}

func (p *controlPlaneResize) getDownsizeMetricsDataStats(ctx context.Context, log *logrus.Entry) ([]MetricData, error) {
	stats := []MetricData{}
	requests := []*http.Request{}
	promQueries := GetDownsizeMetricAvailabilityQueries()
	for _, query := range promQueries {
		req, err := p.createPrometheusQueryRequest(ctx, query.Query)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	resps, err := p.portForward.ForwardHttp(ctx, p.portForwardService, requests)

	for _, r := range resps {
		defer r.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	for i, r := range resps {
		promResp, err := p.parsePrometheusResponse(r)
		if err != nil {
			return nil, err
		}

		log.Infof("Prometheus response: %s", promResp.Data.Result)

		if promResp.Status != "success" {
			return nil, errors.New(ErrMsgPrometheusQueryFailed)
		}

		result, err := p.parseVectorResult(promResp)
		if err != nil {
			return nil, err
		}

		if len(result) != 1 {
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		value, ok := result[0].Value[1].(string)
		if !ok {
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		percent, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		percent = math.Round(percent*100) / 100
		data := MetricData{
			Name:             promQueries[i].Name,
			PercentAvailable: percent,
			Query:            promQueries[i].Query,
		}
		stats = append(stats, data)
	}

	return stats, nil
}

func (c *controlPlaneResize) assessDownsizeRequest(ctx context.Context, log *logrus.Entry) (DownsizeAssessment, error) {
	metricData, err := c.getDownsizeMetricsDataStats(ctx, log)

	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	if err != nil {
		log.Errorf("%s: %v", ErrMsgMetricDataUnavailable, err)

		err = errors.New(ErrMsgMetricDataUnavailable)
		return DownsizeAssessment{
			Err:             errStr,
			MetricDataStats: metricData,
			Proceed:         false,
			Recommendation:  IncompleteDownsizeMsg,
		}, err
	}

	// 1. Ensure there are enough metric data
	for _, data := range metricData {
		if data.PercentAvailable < 80 {
			err := fmt.Errorf(
				"metric %s has only %v%% data points: %s",
				data.Name,
				data.PercentAvailable,
				ErrMsgMetricDataUnavailable,
			)
			return DownsizeAssessment{
				Err:             err.Error(),
				MetricDataStats: metricData,
				Proceed:         false,
				Recommendation:  DoNotProceedDownsizeMsg,
			}, err
		}
	}

	return DownsizeAssessment{
		MetricDataStats: metricData,
		Proceed:         true,
		Recommendation:  ProceedDownsizeMsg,
	}, nil
}

func (p portForwardService) GetPortForwardPodName() string {
	return p.portForwardPodName
}

func (p portForwardService) GetPortForwardNamespace() string {
	return p.portForwardNamespace
}

func (p portForwardService) GetPortForwardPort() string {
	return p.portForwardPort
}
