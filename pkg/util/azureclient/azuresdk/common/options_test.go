package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// trackingReadCloser wraps a ReadCloser and tracks if Close() was called
type trackingReadCloser struct {
	io.ReadCloser
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return t.ReadCloser.Close()
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name             string
		bodyContent      string
		statusCode       int
		err              error
		expectedRetry    bool
		expectedReason   string
		expectBodyClosed bool
	}{
		{
			name:             "http body not read: retry on error without response",
			bodyContent:      "",
			statusCode:       0,
			err:              io.EOF,
			expectedRetry:    true,
			expectedReason:   "should retry when there's an error",
			expectBodyClosed: false,
		},
		{
			name:             "http body not read: no retry on 2xx success",
			bodyContent:      "",
			statusCode:       http.StatusOK,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "should not retry on 2xx status codes",
			expectBodyClosed: false,
		},
		{
			name: "http body read: retry on AuthorizationFailed",
			bodyContent: `{
				"error": {
					"code": "AuthorizationFailed",
					"message": "The client does not have authorization to perform action"
				}
			}`,
			statusCode:       http.StatusForbidden,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry when AuthorizationFailed appears in response body",
			expectBodyClosed: true,
		},
		{
			name: "http body read: retry on AADSTS7000215",
			bodyContent: `{
				"error": "invalid_client",
				"error_description": "AADSTS7000215: Invalid client secret provided"
			}`,
			statusCode:       http.StatusUnauthorized,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry when AADSTS7000215 appears in response body",
			expectBodyClosed: true,
		},
		{
			name: "http body read: retry on AADSTS7000216",
			bodyContent: `{
				"error": "invalid_request",
				"error_description": "AADSTS7000216: Missing required parameters"
			}`,
			statusCode:       http.StatusBadRequest,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry when AADSTS7000216 appears in response body",
			expectBodyClosed: true,
		},
		{
			name: "http body read: no retry on InvalidParameter",
			bodyContent: `{
				"error": {
					"code": "InvalidParameter",
					"message": "The parameter is invalid"
				}
			}`,
			statusCode:       http.StatusBadRequest,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "should not retry on 4xx errors without special error codes",
			expectBodyClosed: true,
		},
		{
			name:             "http body not read: retry on 429 status code",
			bodyContent:      "",
			statusCode:       http.StatusTooManyRequests,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry on 429 status code (in autorest.StatusCodesForRetry)",
			expectBodyClosed: false,
		},
		{
			name:             "http body not read: retry on 500 status code",
			bodyContent:      "",
			statusCode:       http.StatusInternalServerError,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry on 500 status code (in autorest.StatusCodesForRetry)",
			expectBodyClosed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var tracker *trackingReadCloser

			if tt.err == nil {
				// Wrap body in tracker to verify Close() is called
				tracker = &trackingReadCloser{
					ReadCloser: io.NopCloser(strings.NewReader(tt.bodyContent)),
					closed:     false,
				}
				resp = &http.Response{
					StatusCode: tt.statusCode,
					Body:       tracker,
				}
			}

			result := shouldRetry(resp, tt.err)
			if result != tt.expectedRetry {
				t.Errorf("shouldRetry() = %v, want %v (%s)", result, tt.expectedRetry, tt.expectedReason)
			}

			// Verify original body was closed when expected (i.e., when body was read and replaced)
			if tracker != nil {
				if tt.expectBodyClosed && !tracker.closed {
					t.Errorf("shouldRetry() did not close the original body when it should have - this will leak HTTP connections")
				}
				if !tt.expectBodyClosed && tracker.closed {
					t.Errorf("shouldRetry() closed the body unexpectedly - this could break SDK error handling")
				}
			}

			// Verify body is still readable and unchanged when it was inspected (regression test)
			// This ensures the body restoration logic works correctly
			if resp != nil && tt.bodyContent != "" && tt.expectBodyClosed {
				bodyAfter, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("failed to read body after shouldRetry: %v", err)
				}
				// Normalize whitespace for comparison
				originalNormalized := strings.Join(strings.Fields(tt.bodyContent), "")
				afterNormalized := strings.Join(strings.Fields(string(bodyAfter)), "")
				if originalNormalized != afterNormalized {
					t.Errorf("body content changed after shouldRetry:\nwant: %s\ngot:  %s", tt.bodyContent, string(bodyAfter))
				}
			}
		})
	}
}
