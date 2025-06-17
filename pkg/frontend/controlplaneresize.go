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
	ErrMsgMetricDataUnavailable           = "the required metric data is unavailable"
	ErrMsgNotEnoughMetricData             = "not enough metric data points"
	ErrMsgUnknownPrometheusResponse       = "unknown prometheus response"
	ErrMsgInvalidPrometheusResponse       = "invalid prometheus response"
	ErrMsgPrometheusQueryFailed           = "prometheus query failed"
	ErrMsgPrometheusRequestCreationFailed = "prometheus request creation failed"
	ErrMsgPrometheusQueryUnexpected       = "prometheus query returned unexpected type or value"
	ErrMsgInvalidDownsizeRequest          = "invalid downsize request"
	ErrMsgCondition1                      = "the current average memory usage for the last 2 weeks does not fit in the target provisioned memory"
	ErrMsgCondition2                      = "the current average CPU usage for the last 2 weeks does not fit in the target provisioned CPU"
	ErrMsgCondition3                      = "the average memory usage over 2 weeks is close to 60% of the target provisioned memory, and the total average memory available trend is decreasing"
	ErrMsgCondition4                      = "the average CPU usage over 2 weeks is close to 60% of the target provisioned CPU, and the total average CPU usage available trend is decreasing"

	ProceedDownsizeMsg      = "assessment successful, proceed with downsize"
	DoNotProceedDownsizeMsg = "assessment unsuccessful, do not proceed with downsize"
	IncompleteDownsizeMsg   = "assessment incomplete due to errors"
)

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

type AlertData struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Severity string `json:"severity"`
}

type DownsizeAssessment struct {
	Proceed         bool         `json:"proceed"`
	Recommendation  string       `json:"recommendation"`
	Err             string       `json:"error,omitempty"`
	MetricDataStats []MetricData `json:"metricDataStats,omitempty"`
	FiringAlerts    []AlertData  `json:"firingalerts"`
}

type DownsizeRequest interface {
	// Azure-advertised instance memory size
	GetInstanceMemorySizeGB() int64
	GetInstanceCPUSize() int64
	// Azure-advertised instance memory size
	GetTargetInstanceMemorySizeGB() int64
	GetTargetInstanceCPUSize() int64
	GetNumControlPlaneNodes() int64
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

func (p *controlPlaneResize) parsePrometheusVectorResult(resp *http.Response) ([]VectorResult, error) {
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	var result PrometheusResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	var vector []VectorResult
	if result.Data.ResultType == "vector" {
		json.Unmarshal(result.Data.Result, &vector)
		return vector, nil
	}

	return nil, fmt.Errorf("result type %s is not a vector: %w", result.Data.ResultType, ErrMsgInvalidPrometheusResponse)
}

func (p *controlPlaneResize) getFiringAlerts(ctx context.Context, log *logrus.Entry) ([]AlertData, error) {
	alerts := []AlertData{}
	query := GetFiringAlertsQuery()

	req, err := p.createPrometheusQueryRequest(ctx, query.Query)
	if err != nil {
		log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
		return nil, errors.New(ErrMsgPrometheusRequestCreationFailed)
	}

	resps, err := p.portForward.ForwardHttp(ctx, p.portForwardService, []*http.Request{req})

	var r *http.Response
	if len(resps) > 0 {
		r = resps[0]
		defer r.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	vectorResult, err := p.parsePrometheusVectorResult(r)

	vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
	if err != nil {
		log.Errorf("Error marshaling vector result: %v", err)
	}

	log.Infof("vector result: %s", vectorResultJson)

	if len(vectorResult) < 1 {
		return alerts, nil
	}

	for _, alert := range vectorResult {
		alertName := alert.Metric["alertname"]
		alertState := alert.Metric["alertstate"]
		alertSeverity := alert.Metric["severity"]
		// no need to check for the value since the query itself
		// guarantees the returne values are firing alerts.

		alerts = append(alerts, AlertData{
			Name:     alertName,
			State:    alertState,
			Severity: alertSeverity,
		})

	}
	return alerts, nil
}

func (p *controlPlaneResize) getDownsizeMetricsDataStats(ctx context.Context, log *logrus.Entry) ([]MetricData, error) {
	stats := []MetricData{}
	requests := []*http.Request{}
	promQueries := GetDownsizeMetricAvailabilityQueries()
	for _, query := range promQueries {
		req, err := p.createPrometheusQueryRequest(ctx, query.Query)
		if err != nil {
			log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
			return nil, errors.New(ErrMsgPrometheusRequestCreationFailed)
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
		vectorResult, err := p.parsePrometheusVectorResult(r)

		vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
		if err != nil {
			log.Errorf("Error marshaling vector result: %v", err)
		}

		log.Infof("vector result: %s", vectorResultJson)

		if len(vectorResult) < 1 {
			// Expect the query to return 1 vector
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		value, ok := vectorResult[0].Value[1].(string)
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

func (p *controlPlaneResize) checkForDownsizeConditions(ctx context.Context, downsizeRequest DownsizeRequest, log *logrus.Entry) error {
	memSizeFactor := (downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetTargetInstanceMemorySizeGB()) /
		(downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetInstanceMemorySizeGB())
	cpuSizeFactor := (downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetTargetInstanceCPUSize()) /
		(downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetInstanceCPUSize())

	condition1Query := GetDownsizeCondition1Query(memSizeFactor)
	condition2Query := GetDownsizeCondition2Query(cpuSizeFactor)
	condition3Query := GetDownsizeCondition3Query(memSizeFactor)
	condition4Query := GetDownsizeCondition4Query(cpuSizeFactor)

	// fmt.Printf("\n%s\n", condition1Query.Query)
	// fmt.Printf("\n%s\n", condition2Query.Query)
	// fmt.Printf("\n%s\n", condition3Query.Query)
	// fmt.Printf("\n%s\n", condition4Query.Query)

	type item struct {
		query PrometheusQuery
		err   error
	}

	requests := []*http.Request{}
	promQueries := []item{
		{condition1Query, errors.New(ErrMsgCondition1)},
		{condition2Query, errors.New(ErrMsgCondition2)},
		{condition3Query, errors.New(ErrMsgCondition3)},
		{condition4Query, errors.New(ErrMsgCondition4)},
	}

	for _, q := range promQueries {
		req, err := p.createPrometheusQueryRequest(ctx, q.query.Query)
		if err != nil {
			log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
			return errors.New(ErrMsgPrometheusRequestCreationFailed)
		}
		requests = append(requests, req)
	}

	resps, err := p.portForward.ForwardHttp(ctx, p.portForwardService, requests)

	for _, r := range resps {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	for i, r := range resps {

		fmt.Printf("\nstatus: %s, body: %s\n", r.Status, r.Body)

		vectorResult, err := p.parsePrometheusVectorResult(r)

		vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
		if err != nil {
			log.Errorf("Error marshaling vector result: %v", err)
		}

		log.Infof("vector result: %s", vectorResultJson)

		if len(vectorResult) < 1 {
			// Expect the query to return 1 vector
			return errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		value, ok := vectorResult[0].Value[1].(string)
		if !ok {
			return errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		isPassed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		if isPassed == 0 {
			// fail immediately when at least 1 condition is not satisfied
			return promQueries[i].err
		}
	}

	return nil
}

func (c *controlPlaneResize) assessDownsizeRequest(ctx context.Context, downsizeRequest DownsizeRequest, log *logrus.Entry) (DownsizeAssessment, error) {
	var err error

	// Get firing alerts
	firingAlerts, err := c.getFiringAlerts(ctx, log)
	if err != nil {
		log.Warnf("%v: %v", ErrMsgMetricDataUnavailable, err)
	}

	metricData, err := c.getDownsizeMetricsDataStats(ctx, log)

	// var errStr string
	// if err != nil {
	// 	errStr = err.Error()
	// }

	// if err != nil {
	// 	log.Errorf("%s: %v", ErrMsgMetricDataUnavailable, err)

	// 	err = errors.New(ErrMsgMetricDataUnavailable)
	// 	return DownsizeAssessment{
	// 		Err:             errStr,
	// 		MetricDataStats: metricData,
	// 		Proceed:         false,
	// 		Recommendation:  IncompleteDownsizeMsg,
	// 		FiringAlerts:    firingAlerts,
	// 	}, err
	// }

	// // 1. Ensure there are enough metric data
	// for _, data := range metricData {
	// 	if data.PercentAvailable < 80 {
	// 		err := fmt.Errorf(
	// 			"metric %s has only %v%% data points: %s",
	// 			data.Name,
	// 			data.PercentAvailable,
	// 			ErrMsgMetricDataUnavailable,
	// 		)
	// 		return DownsizeAssessment{
	// 			Err:             err.Error(),
	// 			MetricDataStats: metricData,
	// 			Proceed:         false,
	// 			Recommendation:  DoNotProceedDownsizeMsg,
	// 			FiringAlerts:    firingAlerts,
	// 		}, err
	// 	}
	// }

	// 2. Ensure all downsize conditions are satisfied
	err = c.checkForDownsizeConditions(ctx, downsizeRequest, log)
	if err != nil {
		return DownsizeAssessment{
			Err:             err.Error(),
			MetricDataStats: metricData,
			Proceed:         false,
			Recommendation:  DoNotProceedDownsizeMsg,
			FiringAlerts:    firingAlerts,
		}, err
	}

	return DownsizeAssessment{
		MetricDataStats: metricData,
		Proceed:         true,
		Recommendation:  ProceedDownsizeMsg,
		FiringAlerts:    firingAlerts,
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
