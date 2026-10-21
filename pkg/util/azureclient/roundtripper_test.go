package azureclient

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
				"response_status_code": "0",
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
				Header:        http.Header{correlationIdHeader: []string{"the_correlation_request_id"}},
				ContentLength: int64(10),
			},
			expectedSubfields: logrus.Fields{
				"response_status_code": http.StatusOK,
				"correlation_id":       "the_correlation_request_id",
				"content_length":       int64(10),
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

			dur, ok := l.Data[durationMilliseconds].(int64)
			if !ok || dur < 0 {
				t.Fatalf("expected non-negative duration_milliseconds, got %v", l.Data[durationMilliseconds])
			}
		})
	}
}

func TestAzureErrorCode(t *testing.T) {
	for _, tt := range []struct {
		name     string
		body     string
		wantCode string
	}{
		{
			name:     "nil body",
			body:     "",
			wantCode: "",
		},
		{
			name:     "nested error.code",
			body:     `{"error":{"code":"ConflictingConcurrentWriteNotAllowed","message":"Please retry later."}}`,
			wantCode: "ConflictingConcurrentWriteNotAllowed",
		},
		{
			name:     "top-level code",
			body:     `{"code":"TooManyRequests","message":"Rate limited."}`,
			wantCode: "TooManyRequests",
		},
		{
			name:     "nested takes priority over top-level",
			body:     `{"error":{"code":"Inner"},"code":"Outer"}`,
			wantCode: "Inner",
		},
		{
			name:     "invalid JSON returns empty",
			body:     `not json`,
			wantCode: "",
		},
		{
			name:     "body restored after parse",
			body:     `{"error":{"code":"ScopeLocked","message":"locked."}}`,
			wantCode: "ScopeLocked",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var res *http.Response
			if tt.body == "" && tt.name == "nil body" {
				res = &http.Response{Body: nil}
			} else {
				res = &http.Response{Body: io.NopCloser(strings.NewReader(tt.body))}
			}

			got := azureErrorCode(res)
			if got != tt.wantCode {
				t.Errorf("azureErrorCode() = %q, want %q", got, tt.wantCode)
			}

			// Verify body was restored and is re-readable.
			if tt.name == "body restored after parse" && res.Body != nil {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("re-reading body: %v", err)
				}
				if string(b) != tt.body {
					t.Errorf("body not restored: got %q, want %q", string(b), tt.body)
				}
			}
		})
	}
}
