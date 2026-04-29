package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

// transientRetryBackoff: 4 attempts, worst-case ~115.5s total sleep. Mutated in tests; no t.Parallel().
var transientRetryBackoff = wait.Backoff{
	Steps:    4,
	Duration: 15 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
	Cap:      60 * time.Second,
}

// retryable wraps an ARM write with transient retry.
func (m *manager) retryable(desc string, f func() error) error {
	return retry.OnError(transientRetryBackoff, m.isRetryable(desc), f)
}

func (m *manager) isRetryable(desc string) func(error) bool {
	return func(err error) bool {
		if azureerrors.IsRetryableError(err) {
			m.log.Warnf("error on %s, retrying: %v", desc, err)
			return true
		}
		return false
	}
}

// retryableDelete wraps an ARM delete with transient retry, treating 404 as success.
func (m *manager) retryableDelete(desc string, f func() error) error {
	return m.retryable(desc, func() error {
		err := f()
		if azureerrors.IsNotFoundError(err) {
			return nil
		}
		return err
	})
}
