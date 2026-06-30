package arm

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

// TransientBackoff is the retry schedule for ARM write operations when no Retry-After
// header is present. It is a var so tests can override it; tests that mutate it must
// not call t.Parallel().
var TransientBackoff = wait.Backoff{
	Steps:    4,
	Duration: 15 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
	Cap:      60 * time.Second,
}

// Retryable wraps f with transient ARM retry and logs each retry attempt.
// If the error carries a Retry-After header, that duration is used as the sleep;
// otherwise TransientBackoff governs the schedule.
func Retryable(ctx context.Context, f func() error, log *logrus.Entry, desc string) error {
	b := TransientBackoff
	steps := b.Steps
	var lastErr error
	for i := 0; i < steps; i++ {
		lastErr = f()
		if lastErr == nil {
			return nil
		}
		if !azureerrors.IsRetryableError(lastErr) {
			return lastErr
		}
		if i == steps-1 {
			break
		}
		sleep := b.Step()
		if d := retryAfterDuration(lastErr); d > 0 {
			sleep = d
		}
		log.WithField("retry_after", sleep.Seconds()).Warnf("error on %s, retrying: %v", desc, lastErr)
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

// RetryableDelete wraps f with transient ARM retry and logs each attempt, treating 404 as success.
func RetryableDelete(ctx context.Context, f func() error, log *logrus.Entry, desc string) error {
	return Retryable(ctx, func() error {
		err := f()
		if azureerrors.IsStatusNotFoundError(err) {
			return nil
		}
		return err
	}, log, desc)
}

// retryAfterDuration returns the Retry-After header value from an ARM error, if present.
// Supports integer seconds only; HTTP-date form is not handled.
func retryAfterDuration(err error) time.Duration {
	var h string
	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) && responseErr.RawResponse != nil {
		h = responseErr.RawResponse.Header.Get("Retry-After")
	}
	if h == "" {
		var detailedErr autorest.DetailedError
		if errors.As(err, &detailedErr) && detailedErr.Response != nil {
			h = detailedErr.Response.Header.Get("Retry-After")
		}
	}
	if h == "" {
		return 0
	}
	secs, parseErr := strconv.Atoi(h)
	if parseErr != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}
