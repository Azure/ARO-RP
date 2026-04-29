package common

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	testhttp "github.com/Azure/ARO-RP/test/util/http/server"
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
		retryAfter       string
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
			name:             "no retry on 403 AuthorizationFailed without Retry-After",
			bodyContent:      `{"error":{"code":"AuthorizationFailed","message":"not authorized"}}`,
			statusCode:       http.StatusForbidden,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "shouldRetry does not read body; semantic retries handled by IsRetryableError",
			expectBodyClosed: false,
		},
		{
			name:             "no retry on 401 AADSTS7000215 without Retry-After",
			bodyContent:      `{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret provided"}`,
			statusCode:       http.StatusUnauthorized,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "shouldRetry does not read body; AADSTS semantic retries handled by IsRetryableError",
			expectBodyClosed: false,
		},
		{
			name:             "no retry on 400 without Retry-After",
			bodyContent:      `{"error":{"code":"InvalidParameter","message":"The parameter is invalid"}}`,
			statusCode:       http.StatusBadRequest,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "should not retry on 4xx errors without Retry-After",
			expectBodyClosed: false,
		},
		{
			name:             "http body not read: retry on 409 with Retry-After header",
			bodyContent:      "",
			statusCode:       http.StatusConflict,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry on 409 with Retry-After header",
			expectBodyClosed: false,
			retryAfter:       "5",
		},
		{
			name: "http body not read: retry on 409 with Retry-After header even when body is non-empty",
			bodyContent: `{
				"error": {
					"code": "ScopeLocked",
					"message": "The scope is locked."
				}
			}`,
			statusCode:       http.StatusConflict,
			err:              nil,
			expectedRetry:    true,
			expectedReason:   "should retry on 409 with Retry-After header regardless of body content",
			expectBodyClosed: false,
			retryAfter:       "5",
		},
		{
			name:             "no retry on 409 without Retry-After header",
			bodyContent:      `{"error":{"code":"ScopeLocked","message":"The scope is locked."}}`,
			statusCode:       http.StatusConflict,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "should not retry on 409 without Retry-After; semantic retry via arm.Retryable()/IsRetryableError",
			expectBodyClosed: false,
		},
		{
			name:             "no retry on 409 with Please retry later body but no Retry-After header",
			bodyContent:      `{"error":{"code":"ConflictingConcurrentWriteNotAllowed","message":"Please retry later."}}`,
			statusCode:       http.StatusConflict,
			err:              nil,
			expectedRetry:    false,
			expectedReason:   "shouldRetry does not read body; semantic retry via arm.Retryable()/IsRetryableError",
			expectBodyClosed: false,
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
					Header:     http.Header{},
					Body:       tracker,
				}
				if tt.retryAfter != "" {
					resp.Header.Set("Retry-After", tt.retryAfter)
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
		})
	}
}

// TestRetryOptionsIntegration verifies that shouldRetry is correctly wired into
// the azcore SDK retry pipeline, not just that shouldRetry() works in isolation.
func TestRetryOptionsIntegration(t *testing.T) {
	tests := []struct {
		name         string
		firstCode    int
		retryAfter   string // Retry-After header value; empty means no header
		firstBody    []byte
		wantRequests int
		wantCode     int
	}{
		{
			// shouldRetry returns true → azcore retries → second request gets 200.
			// Proves the callback is actually wired into the SDK retry loop.
			// Note: Retry-After "0" satisfies shouldRetry's non-empty header check,
			// but the SDK treats 0 as absent for delay purposes and uses calcDelay.
			// The retry wiring is what this test proves, not Retry-After delay behavior.
			name:         "409 with Retry-After header: SDK retries and succeeds",
			firstCode:    http.StatusConflict,
			retryAfter:   "0",
			wantRequests: 2,
			wantCode:     http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, close := testhttp.NewServer()
			defer close()

			respOpts := []testhttp.ResponseOption{testhttp.WithStatusCode(tt.firstCode)}
			if tt.retryAfter != "" {
				respOpts = append(respOpts, testhttp.WithHeader("Retry-After", tt.retryAfter))
			}
			if tt.firstBody != nil {
				respOpts = append(respOpts, testhttp.WithBody(tt.firstBody))
			}
			srv.AppendResponse(respOpts...)
			srv.AppendResponse() // 200 OK fallback; only consumed when SDK retries

			retryOpts := RetryOptions
			retryOpts.MaxRetries = 3
			// Use a small positive value: azcore setDefaults replaces 0 with 800ms/60s,
			// but leaves any positive value untouched. 1ms gives instant-for-CI retries.
			retryOpts.RetryDelay = time.Millisecond
			retryOpts.MaxRetryDelay = time.Millisecond

			pl := runtime.NewPipeline("test", "v1.0.0", runtime.PipelineOptions{}, &policy.ClientOptions{
				Transport: srv,
				Retry:     retryOpts,
			})

			req, err := runtime.NewRequest(context.Background(), http.MethodGet, srv.URL())
			if err != nil {
				t.Fatal(err)
			}

			resp, err := pl.Do(req)
			if err != nil {
				t.Fatalf("unexpected pipeline error: %v", err)
			}
			defer resp.Body.Close()

			if got := srv.Requests(); got != tt.wantRequests {
				t.Errorf("server got %d requests, want %d", got, tt.wantRequests)
			}
			if resp.StatusCode != tt.wantCode {
				t.Errorf("final status = %d, want %d", resp.StatusCode, tt.wantCode)
			}
		})
	}
}
