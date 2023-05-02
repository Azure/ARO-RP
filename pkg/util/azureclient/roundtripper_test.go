package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

// fieldsContainsSubfields returns true if all the keys of subfields are contained in fields and the value
// for each key is the same in fields and in subfields.
func fieldsContainsSubfields(fields logrus.Fields, subfields logrus.Fields) (containsSubfields bool) {
	for subfieldsKey := range subfields {
		fieldsValue, subfieldsKeyIsInFields := fields[subfieldsKey]
		if !subfieldsKeyIsInFields {
			fmt.Printf("fields does not contain subfieldsKey %v\n", subfieldsKey)
			return false
		}

		if fieldsValue != subfields[subfieldsKey] {
			fmt.Printf("fields value %v != subfields value %v\n", fieldsValue, subfields[subfieldsKey])
			return false
		}
	}
	return true
}

func TestUpdateCorrelationDataAndEnrichLogWithRequest(t *testing.T) {
	type testCase struct {
		name              string
		correlationData   *api.CorrelationData
		req               *http.Request
		expectedSubfields logrus.Fields
	}

	startTime := time.Now()
	url, err := url.Parse("https://example.com/foo%2fbar")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []testCase{
		{
			name: "updateCorrelationDataAndEnrichLogWithReq returns appropriate logrus.Entry",
			correlationData: &api.CorrelationData{
				ClientRequestID: "ClientRequestID",
				CorrelationID:   "CorrelationID",
				RequestID:       "random_request_id",
				OperationID:     "random_operation_id",
			},
			req: &http.Request{
				URL: url,
			},
			expectedSubfields: logrus.Fields{
				"client_request_id": "ClientRequestID",
				"correlation_id":    "CorrelationID",
				"request_time":      startTime,
				"request_id":        "random_request_id",
				"LOGKIND":           "outboundRequests",
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			baseLogger := utillog.GetLogger()
			l := updateCorrelationDataAndEnrichLogWithRequest(tc.correlationData, baseLogger, startTime, tc.req)
			if tc.correlationData.RequestTime != startTime {
				t.Fatal("wrong RequestTime registration")
			}

			operationId, ok := l.Data["operation_id"].(string)
			if !ok {
				t.Fatal("generated operation id is not of string type")
			}

			if operationId == "" {
				t.Fatal("generated operationId should not be empty")
			}

			if !fieldsContainsSubfields(l.Data, tc.expectedSubfields) {
				t.Fatal("logrus entry does not contain all expectedSubfields")
			}
		})
	}
}

func TestUpdateCorrelationDataAndEnrichLogWithResponse(t *testing.T) {
	type testCase struct {
		name                         string
		res                          *http.Response
		correlationData              *api.CorrelationData
		expecetdCorrelationIDUpdated string
		expectedSubfields            logrus.Fields
	}

	startTime := time.Now()
	testcases := []testCase{
		{
			name: "updateCorrelationDataAndEnrichLogWithResponse returns appropriate logrus.Entry when response is nil",
			res:  nil,
			correlationData: &api.CorrelationData{
				ClientRequestID: "ClientRequestID",
				CorrelationID:   "CorrelationID",
				RequestID:       "random_request_id",
				OperationID:     "random_operation_id",
				RequestTime:     startTime,
			},
			expecetdCorrelationIDUpdated: "CorrelationID",
			expectedSubfields: logrus.Fields{
				"response_status_code":   "0",
				"contentLength":          "-1",
				"durationInMilliseconds": time.Since(startTime).Milliseconds(),
			},
		},
		{
			name: "updateCorrelationDataAndEnrichLogWithResponse returns appropriate logrus.Entry when response is not nil",
			correlationData: &api.CorrelationData{
				ClientRequestID: "ClientRequestID",
				CorrelationID:   "CorrelationID",
				RequestID:       "random_request_id",
				OperationID:     "random_operation_id",
				RequestTime:     startTime,
			},
			expecetdCorrelationIDUpdated: "the_correlation_request_id",
			res: &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"X-Ms-Correlation-Request-Id": []string{"the_correlation_request_id"}},
				ContentLength: int64(10),
			},
			expectedSubfields: logrus.Fields{
				"response_status_code": http.StatusOK,
				"correlation_id":       "the_correlation_request_id",
				"contentLength":        int64(10),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			baseLogger := utillog.GetLogger()

			l := updateCorrelationDataAndEnrichLogWithResponse(tc.correlationData, baseLogger, tc.res, startTime)

			if tc.correlationData.CorrelationID != tc.expecetdCorrelationIDUpdated {
				t.Fatal("correlationData.CorrelationID not updated properly")
			}

			if !fieldsContainsSubfields(l.Data, tc.expectedSubfields) {
				t.Fatal("logrus entry does not contain all expectedSubfields")
			}
		})
	}
}
