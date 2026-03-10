package kubernetes

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"
)

// RetryDelay is the delay between Retry attempts. It is a variable rather than
// a constant so that tests can override it to zero.
var RetryDelay = 2 * time.Second

// Retry retries fn up to maxAttempts times with RetryDelay between attempts.
// Returns nil on first success, ctx.Err() on cancellation, or the last error
// if all attempts are exhausted. If maxAttempts <= 0, fn is called once.
func Retry(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	var err error
	for i := range maxAttempts {
		err = fn()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if i < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(RetryDelay):
			}
		}
	}
	return err
}
