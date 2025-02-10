package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

const (
	// outboundRequests is the table name we configure in Geneva
	// to send logs of outgoing requests from ARO-RP to ARM.
	// https://docs.google.com/document/d/1RbnKKPNjw7kJZeR-2me4eu
	outboundRequests = "outboundRequests"

	responseCode         = "response_status_code"
	contentLength        = "content_length"
	durationMilliseconds = "duration_milliseconds"
	correlationIdHeader  = "X-Ms-Correlation-Request-Id"
)

type PolicyFunc func(req *policy.Request) (*http.Response, error)

func (p PolicyFunc) Do(req *policy.Request) (*http.Response, error) {
	return p(req)
}

var _ policy.Policy = PolicyFunc(nil)

func NewLoggingPolicy() policy.Policy {
	return PolicyFunc(func(req *policy.Request) (*http.Response, error) {
		return loggingRoundTripper(req.Raw(), req.Next)
	})
}

func loggingRoundTripper(req *http.Request, next func() (*http.Response, error)) (*http.Response, error) {
	correlationData := api.GetCorrelationDataFromCtx(req.Context())
	if correlationData == nil {
		correlationData = api.CreateCorrelationDataFromReq(req)
	} else if correlationData.CorrelationID != "" {
		req.Header.Set(correlationIdHeader, correlationData.CorrelationID)
	}

	requestTime := time.Now()
	l := updateCorrelationDataAndEnrichLogWithRequest(correlationData, utillog.GetLogger(), requestTime, req)

	l.Info("HttpRequestStart")

	res, err := next()

	l = updateCorrelationDataAndEnrichLogWithResponse(correlationData, l, res, requestTime)
	l.Info("HttpRequestEnd")

	return res, err
}

// updateCorrelationDataAndEnrichLogWithRequest receives a non nil correlationData and updates the request time.
// It also returns a new logrus entry updated with the new field for the LOGKIND entry (custom DGrep table for outgoing requests).
func updateCorrelationDataAndEnrichLogWithRequest(correlationData *api.CorrelationData, l *logrus.Entry, requestTime time.Time, req *http.Request) *logrus.Entry {
	correlationData.RequestTime = requestTime

	l = utillog.EnrichWithCorrelationData(l, correlationData)
	l = l.WithFields(logrus.Fields{
		"request_URL": req.URL.Host,
		"LOGKIND":     outboundRequests,
	})

	return l
}

func updateCorrelationDataAndEnrichLogWithResponse(correlationData *api.CorrelationData, l *logrus.Entry, res *http.Response, requestTime time.Time) *logrus.Entry {
	if res == nil {
		return l.WithFields(logrus.Fields{
			responseCode:         "0",
			durationMilliseconds: time.Since(requestTime).Milliseconds(),
		})
	}

	correlationData.CorrelationID = res.Header.Get(correlationIdHeader)
	l = utillog.EnrichWithCorrelationData(l, correlationData)

	return l.WithFields(logrus.Fields{
		responseCode:         res.StatusCode,
		contentLength:        res.ContentLength,
		durationMilliseconds: time.Since(requestTime).Milliseconds(),
	})
}
