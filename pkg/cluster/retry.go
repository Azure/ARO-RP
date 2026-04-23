package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
)

// transientRetryBackoff is the backoff for transient ARM errors. Steps=4 gives 4 attempts; inter-retry sleeps are non-interruptible (time.Sleep), worst case ≈115.5s. Mutated in tests; no t.Parallel().
var transientRetryBackoff = wait.Backoff{
	Steps:    4,
	Duration: 15 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
	Cap:      60 * time.Second,
}

// isRetryable returns a retry predicate that logs a warning and returns true for transient ARM errors.
func (m *manager) isRetryable(desc string) func(error) bool {
	return func(err error) bool {
		if azureerrors.IsRetryableError(err) {
			m.log.Warnf("transient error on %s, retrying: %v", desc, err)
			return true
		}
		return false
	}
}
