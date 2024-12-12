package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"time"

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

func NewCustomRoundTripper(next http.RoundTripper) http.RoundTripper {
	return &customRoundTripper{
		next: next,
	}
}

type customRoundTripper struct {
	next http.RoundTripper
}

func (crt *customRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	correlationData := api.GetCorrelationDataFromCtx(req.Context())
	if correlationData == nil {
		correlationData = api.CreateCorrelationDataFromReq(req)
	} else if correlationData.CorrelationID != "" {
		req.Header.Set(correlationIdHeader, correlationData.CorrelationID)
	}

	requestTime := time.Now()
	l := updateCorrelationDataAndEnrichLogWithRequest(correlationData, utillog.GetLogger(), requestTime, req)

	l.Info("HttpRequestStart")

	res, err := crt.next.RoundTrip(req)

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
