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
	ErrMsgUnexpectedPrometheusResponse    = "unexpected prometheus response"
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

// cpResizeAssessment represents a utility for assessing control-plane resizes.
type cpResizeAssessment struct {
	// portForward provides a k8s port-forwarded sessions to pods.
	portForward adminactions.PortForwardActions
	// portForwardService is typically the pod that being portforwarded to.
	portForwardService adminactions.PortForwardService
}

type portForwardService struct {
	podName      string
	podNamespace string
	podPort      string
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

// A representation of a downsize assessment result given a downsize request
type DownsizeAssessment struct {
	Proceed         bool         `json:"proceed"`
	Recommendation  string       `json:"recommendation"`
	Err             string       `json:"error,omitempty"`
	MetricData      []MetricData `json:"metricData,omitempty"`
	FiringAlerts    []AlertData  `json:"firingalerts"`
	FailedCondition int          `json:"failedcondition"`
}

type DownsizeRequest interface {
	// Azure-advertised instance memory size
	GetInstanceMemorySizeGB() int64
	// Azure-advertised instance CPU size
	GetInstanceCPUSize() int64
	// Azure-advertised instance memory size
	GetTargetInstanceMemorySizeGB() int64
	// Azure-advertised instance CPU size
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

func newCPResizeAssessment(portForward adminactions.PortForwardActions) *cpResizeAssessment {
	portFwdSvc := portForwardService{
		podName:      "prometheus-k8s-0",
		podNamespace: "openshift-monitoring",
		podPort:      "9090",
	}
	return &cpResizeAssessment{
		portForward:        portForward,
		portForwardService: portFwdSvc,
	}
}

// Creates and returns an http request for a prometheus query api.
func (c *cpResizeAssessment) createPrometheusQueryRequest(ctx context.Context, query string) (*http.Request, error) {
	promURL := fmt.Sprintf(
		"http://%s.%s.svc:%s/api/v1/query",
		c.portForwardService.GetPodName(),
		c.portForwardService.GetPodNamespace(),
		c.portForwardService.GetPodPort(),
	)
	params := url.Values{}
	params.Add("query", query)
	fullURL := fmt.Sprintf("%s?%s", promURL, params.Encode())
	return http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
}

// Parses and returns a prometheus http response into a prometheus vector.
func (c *cpResizeAssessment) parsePrometheusVectorResult(resp *http.Response) ([]VectorResult, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s, failed to parse body: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	var result PrometheusResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("%s, failed to unmarshal body: %v", ErrMsgUnknownPrometheusResponse, err)
	}

	var vector []VectorResult
	if result.Data.ResultType == "vector" {
		json.Unmarshal(result.Data.Result, &vector)
		return vector, nil
	}

	return nil, fmt.Errorf("result type %s is not a vector: %w", result.Data.ResultType, ErrMsgUnexpectedPrometheusResponse)
}

// Returns the firing alerts in a cluster's prometheus
func (c *cpResizeAssessment) getFiringAlerts(ctx context.Context, log *logrus.Entry) ([]AlertData, error) {
	alerts := []AlertData{}
	query := GetFiringAlertsQuery()

	req, err := c.createPrometheusQueryRequest(ctx, query.Query)
	if err != nil {
		log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
		return nil, errors.New(ErrMsgPrometheusRequestCreationFailed)
	}

	resps, err := c.portForward.ForwardHttp(ctx, c.portForwardService, []*http.Request{req})

	for _, r := range resps {
		defer r.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	r := resps[0]
	vectorResult, err := c.parsePrometheusVectorResult(r)

	vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
	if err != nil {
		log.Errorf("Error marshaling vector result: %v", err)
	}

	log.Infof("vector result: %s", vectorResultJson)

	for _, alert := range vectorResult {
		alertName := alert.Metric["alertname"]
		alertState := alert.Metric["alertstate"]
		alertSeverity := alert.Metric["severity"]
		alerts = append(alerts, AlertData{
			Name:     alertName,
			State:    alertState,
			Severity: alertSeverity,
		})

	}
	return alerts, nil
}

// Returns information about prometheus metrics to be used for a downsize assessment
func (c *cpResizeAssessment) getDownsizeMetricsData(ctx context.Context, log *logrus.Entry) ([]MetricData, error) {
	data := []MetricData{}
	requests := []*http.Request{}
	promQueries := GetDownsizeMetricAvailabilityQueries()

	for _, query := range promQueries {
		req, err := c.createPrometheusQueryRequest(ctx, query.Query)
		if err != nil {
			log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
			return nil, errors.New(ErrMsgPrometheusRequestCreationFailed)
		}
		requests = append(requests, req)
	}

	resps, err := c.portForward.ForwardHttp(ctx, c.portForwardService, requests)

	for _, r := range resps {
		defer r.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	for i, r := range resps {
		vectorResult, err := c.parsePrometheusVectorResult(r)

		vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
		if err != nil {
			log.Errorf("Error marshaling vector result: %v", err)
		}

		log.Infof("vector result: %s", vectorResultJson)

		if len(vectorResult) == 0 {
			log.Errorf("Vector result is empty: %v", ErrMsgPrometheusQueryUnexpected)
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		value, ok := vectorResult[0].Value[1].(string)
		if !ok {
			log.Errorf("Vector result value is not a string: %v", ErrMsgPrometheusQueryUnexpected)
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		percent, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errorf("Failed to parse vector result value into an float: %v: %v", ErrMsgPrometheusQueryUnexpected, err)
			return nil, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		percent = math.Round(percent*100) / 100
		newData := MetricData{
			Name:             promQueries[i].Name,
			PercentAvailable: percent,
			Query:            promQueries[i].Query,
		}
		data = append(data, newData)
	}

	return data, nil
}

// Checks for downsize assessment conditions (see SOP) and returns the condition
// number and an error if at least one of the conditions is not met
func (c *cpResizeAssessment) checkForDownsizeConditions(
	ctx context.Context,
	downsizeRequest DownsizeRequest,
	log *logrus.Entry,
) (int, error) {
	memSizeFactor := (downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetTargetInstanceMemorySizeGB()) /
		(downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetInstanceMemorySizeGB())
	cpuSizeFactor := (downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetTargetInstanceCPUSize()) /
		(downsizeRequest.GetNumControlPlaneNodes() * downsizeRequest.GetInstanceCPUSize())

	condition1Query := GetDownsizeCondition1Query(memSizeFactor)
	condition2Query := GetDownsizeCondition2Query(cpuSizeFactor)
	condition3Query := GetDownsizeCondition3Query(memSizeFactor)
	condition4Query := GetDownsizeCondition4Query(cpuSizeFactor)

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
	conditionChecksFailed := 0

	for _, q := range promQueries {
		req, err := c.createPrometheusQueryRequest(ctx, q.query.Query)
		if err != nil {
			log.Errorf("%s: %v", ErrMsgPrometheusRequestCreationFailed, err)
			return conditionChecksFailed, errors.New(ErrMsgPrometheusRequestCreationFailed)
		}
		requests = append(requests, req)
	}

	resps, err := c.portForward.ForwardHttp(ctx, c.portForwardService, requests)

	for _, r := range resps {
		defer r.Body.Close()
	}

	if err != nil {
		return conditionChecksFailed, err
	}

	for i, r := range resps {

		vectorResult, err := c.parsePrometheusVectorResult(r)

		vectorResultJson, err := json.MarshalIndent(vectorResult, "", " ")
		if err != nil {
			log.Errorf("Error marshaling vector result: %v", err)
		}

		log.Infof("vector result: %s", vectorResultJson)

		if len(vectorResult) == 0 {
			log.Errorf("Vector result is empty: %v", ErrMsgPrometheusQueryUnexpected)
			return conditionChecksFailed, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		value, ok := vectorResult[0].Value[1].(string)
		if !ok {
			log.Errorf("Vector result value is not a string: %v", ErrMsgPrometheusQueryUnexpected)
			return conditionChecksFailed, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		isPassed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errorf("Failed to parse vector result value into an float: %v: %v", ErrMsgPrometheusQueryUnexpected, err)
			return conditionChecksFailed, errors.New(ErrMsgPrometheusQueryUnexpected)
		}

		if isPassed == 0 {
			// fail immediately when at least 1 condition is not satisfied
			return i + 1, promQueries[i].err
		}
	}

	return 0, nil
}

func (c *cpResizeAssessment) assessDownsizeRequest(ctx context.Context, downsizeRequest DownsizeRequest, log *logrus.Entry) (DownsizeAssessment, error) {
	var err error

	// Get firing alerts
	firingAlerts, err := c.getFiringAlerts(ctx, log)
	if err != nil {
		log.Warnf("%v: %v", ErrMsgMetricDataUnavailable, err)
	}

	metricData, err := c.getDownsizeMetricsData(ctx, log)

	var errStr string
	if err != nil {
		errStr = err.Error()
	}

	if err != nil {
		log.Errorf("%s: %v", ErrMsgMetricDataUnavailable, err)

		err = errors.New(ErrMsgMetricDataUnavailable)
		return DownsizeAssessment{
			Err:            errStr,
			MetricData:     metricData,
			Proceed:        false,
			Recommendation: IncompleteDownsizeMsg,
			FiringAlerts:   firingAlerts,
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
				Err:            err.Error(),
				MetricData:     metricData,
				Proceed:        false,
				Recommendation: DoNotProceedDownsizeMsg,
				FiringAlerts:   firingAlerts,
			}, err
		}
	}

	// 2. Ensure all downsize conditions are satisfied
	failedCondition, err := c.checkForDownsizeConditions(ctx, downsizeRequest, log)
	if err != nil {
		return DownsizeAssessment{
			Err:             err.Error(),
			MetricData:      metricData,
			Proceed:         false,
			Recommendation:  DoNotProceedDownsizeMsg,
			FiringAlerts:    firingAlerts,
			FailedCondition: failedCondition,
		}, err
	}

	return DownsizeAssessment{
		MetricData:     metricData,
		Proceed:        true,
		Recommendation: ProceedDownsizeMsg,
		FiringAlerts:   firingAlerts,
	}, nil
}

func (p portForwardService) GetPodName() string {
	return p.podName
}

func (p portForwardService) GetPodNamespace() string {
	return p.podNamespace
}

func (p portForwardService) GetPodPort() string {
	return p.podPort
}
