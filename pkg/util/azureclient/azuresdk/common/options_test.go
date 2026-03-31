package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name           string
		resp           *http.Response
		err            error
		expectedRetry  bool
		expectedReason string
	}{
		{
			name:           "retry on error",
			resp:           nil,
			err:            io.EOF,
			expectedRetry:  true,
			expectedReason: "should retry when there's an error",
		},
		{
			name: "no retry on success",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			err:            nil,
			expectedRetry:  false,
			expectedReason: "should not retry on 2xx status codes",
		},
		{
			name: "retry on AuthorizationFailed in response body",
			resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Body: io.NopCloser(strings.NewReader(`{
					"error": {
						"code": "AuthorizationFailed",
						"message": "The client does not have authorization to perform action"
					}
				}`)),
			},
			err:            nil,
			expectedRetry:  true,
			expectedReason: "should retry when AuthorizationFailed appears in response body",
		},
		{
			name: "retry on AADSTS7000215 in response body",
			resp: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body: io.NopCloser(strings.NewReader(`{
					"error": "invalid_client",
					"error_description": "AADSTS7000215: Invalid client secret provided"
				}`)),
			},
			err:            nil,
			expectedRetry:  true,
			expectedReason: "should retry when AADSTS7000215 appears in response body",
		},
		{
			name: "retry on AADSTS7000216 in response body",
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`{
					"error": "invalid_request",
					"error_description": "AADSTS7000216: Missing required parameters"
				}`)),
			},
			err:            nil,
			expectedRetry:  true,
			expectedReason: "should retry when AADSTS7000216 appears in response body",
		},
		{
			name: "no retry on non-retryable 4xx without special error codes",
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(`{
					"error": {
						"code": "InvalidParameter",
						"message": "The parameter is invalid"
					}
				}`)),
			},
			err:            nil,
			expectedRetry:  false,
			expectedReason: "should not retry on 4xx errors without special error codes",
		},
		{
			name: "retry on 429 Too Many Requests",
			resp: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			err:            nil,
			expectedRetry:  true,
			expectedReason: "should retry on 429 status code (in autorest.StatusCodesForRetry)",
		},
		{
			name: "retry on 500 Internal Server Error",
			resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("")),
			},
			err:            nil,
			expectedRetry:  true,
			expectedReason: "should retry on 500 status code (in autorest.StatusCodesForRetry)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.resp, tt.err)
			if result != tt.expectedRetry {
				t.Errorf("shouldRetry() = %v, want %v (%s)", result, tt.expectedRetry, tt.expectedReason)
			}
		})
	}
}
