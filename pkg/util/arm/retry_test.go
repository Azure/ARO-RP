package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
)

func TestRetryable(t *testing.T) {
	origBackoff := TransientBackoff
	TransientBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
	defer func() { TransientBackoff = origBackoff }()

	for _, tt := range []struct {
		name            string
		err             error
		wantRetry       bool
		wantLogMsg      string
		wantRetryAfter  float64 // expected retry_after field value; 0 means check only presence
		checkRetryAfter bool
	}{
		{
			name:       "retryable autorest 429 logs and retries",
			err:        autorest.DetailedError{StatusCode: http.StatusTooManyRequests},
			wantRetry:  true,
			wantLogMsg: "error on test-op, retrying:",
		},
		{
			name:       "retryable azcore 429 logs and retries",
			err:        &azcore.ResponseError{StatusCode: http.StatusTooManyRequests},
			wantRetry:  true,
			wantLogMsg: "error on test-op, retrying:",
		},
		{
			name: "retryable autorest 409+Retry-After uses header duration and logs retry_after",
			err: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Response: &http.Response{
					StatusCode: http.StatusConflict,
					Header:     http.Header{"Retry-After": []string{"1"}},
				},
			},
			wantRetry:       true,
			wantLogMsg:      "error on test-op, retrying:",
			wantRetryAfter:  1.0,
			checkRetryAfter: true,
		},
		{
			name:      "non-retryable error is not retried and not logged",
			err:       errors.New("permanent failure"),
			wantRetry: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logger, hook := logrustest.NewNullLogger()
			log := logger.WithField("test", t.Name())

			calls := 0
			err := Retryable(context.Background(), func() error {
				calls++
				return tt.err
			}, log, "test-op")

			assert.Equal(t, tt.err, err)
			if tt.wantRetry {
				assert.Greater(t, calls, 1, "expected at least one retry")
				assert.NotEmpty(t, hook.Entries)
				assert.Contains(t, hook.LastEntry().Message, tt.wantLogMsg)
				assert.Contains(t, hook.LastEntry().Data, "retry_after")
				if tt.checkRetryAfter {
					assert.InDelta(t, tt.wantRetryAfter, hook.LastEntry().Data["retry_after"].(float64), 0.001)
				}
			} else {
				assert.Equal(t, 1, calls)
				assert.Empty(t, hook.Entries)
			}
		})
	}
}

func TestRetryableDelete(t *testing.T) {
	origBackoff := TransientBackoff
	TransientBackoff = wait.Backoff{Steps: 1, Duration: time.Millisecond}
	defer func() { TransientBackoff = origBackoff }()

	for _, tt := range []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "404 from inner f() is swallowed and nil returned",
			err:     autorest.DetailedError{StatusCode: http.StatusNotFound},
			wantErr: nil,
		},
		{
			name:    "non-404 error propagates unchanged",
			err:     autorest.DetailedError{StatusCode: http.StatusConflict},
			wantErr: autorest.DetailedError{StatusCode: http.StatusConflict},
		},
		{
			name:    "nil error propagates unchanged",
			err:     nil,
			wantErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logrustest.NewNullLogger()
			log := logger.WithField("test", t.Name())
			err := RetryableDelete(context.Background(), func() error {
				return tt.err
			}, log, "test-delete")
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestRetryableDeleteRetryPath(t *testing.T) {
	origBackoff := TransientBackoff
	TransientBackoff = wait.Backoff{Steps: 2, Duration: time.Millisecond, Factor: 2.0}
	defer func() { TransientBackoff = origBackoff }()

	calls := 0
	logger, _ := logrustest.NewNullLogger()
	log := logger.WithField("test", t.Name())
	err := RetryableDelete(context.Background(), func() error {
		calls++
		if calls == 1 {
			return autorest.DetailedError{StatusCode: http.StatusTooManyRequests}
		}
		return nil
	}, log, "test-retry-delete")
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "expected inner function to be called twice: once for transient error, once for success")
}

func TestRetryableContextCancellation(t *testing.T) {
	origBackoff := TransientBackoff
	TransientBackoff = wait.Backoff{Steps: 3, Duration: time.Hour, Factor: 1.0}
	defer func() { TransientBackoff = origBackoff }()

	ctx, cancel := context.WithCancel(context.Background())
	logger, _ := logrustest.NewNullLogger()
	log := logger.WithField("test", t.Name())

	transientErr := &azcore.ResponseError{StatusCode: http.StatusTooManyRequests}
	calls := 0
	cancel() // cancel before first retry sleep
	err := Retryable(ctx, func() error {
		calls++
		return transientErr
	}, log, "test-op")

	assert.Equal(t, 1, calls, "f() should be called once before the cancelled sleep exits")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetryAfterDuration(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want time.Duration
	}{
		{
			name: "azcore response with Retry-After header",
			err: &azcore.ResponseError{
				StatusCode: http.StatusTooManyRequests,
				RawResponse: &http.Response{
					Header: http.Header{"Retry-After": []string{"30"}},
				},
			},
			want: 30 * time.Second,
		},
		{
			name: "autorest DetailedError with Retry-After header",
			err: autorest.DetailedError{
				StatusCode: http.StatusConflict,
				Response: &http.Response{
					Header: http.Header{"Retry-After": []string{"60"}},
				},
			},
			want: 60 * time.Second,
		},
		{
			name: "azcore response without Retry-After header",
			err:  &azcore.ResponseError{StatusCode: http.StatusTooManyRequests},
			want: 0,
		},
		{
			name: "Retry-After value of zero is ignored",
			err: &azcore.ResponseError{
				RawResponse: &http.Response{
					Header: http.Header{"Retry-After": []string{"0"}},
				},
			},
			want: 0,
		},
		{
			name: "Retry-After non-integer value is ignored",
			err: &azcore.ResponseError{
				RawResponse: &http.Response{
					Header: http.Header{"Retry-After": []string{"Thu, 01 Jan 2026 00:00:00 GMT"}},
				},
			},
			want: 0,
		},
		{
			name: "generic error returns zero",
			err:  errors.New("something happened"),
			want: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, retryAfterDuration(tt.err))
		})
	}
}
