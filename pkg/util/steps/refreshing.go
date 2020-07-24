package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

// RetryOnAuthorizationFailedError returns a wrapper Step which will refresh
// `authorizer` if the step returns an Azure AuthorizationError and rerun it.
// The step will be retried until `retryTimeout` is hit. Any other error will be
// returned directly.
func RetryOnAuthorizationFailedError(authorizer refreshable.Authorizer, step Step) authorizationRefreshingStep {
	return authorizationRefreshingStep{
		step:         step,
		authorizer:   authorizer,
		retryTimeout: 10 * time.Minute,
		pollInterval: 10 * time.Second,
	}
}

type authorizationRefreshingStep struct {
	step         Step
	authorizer   refreshable.Authorizer
	retryTimeout time.Duration
	pollInterval time.Duration
}

func (s authorizationRefreshingStep) run(ctx context.Context, log *logrus.Entry) error {
	var retryTimeout time.Duration

	// If it's a condition, absorb its retryTimeout.
	c, isCondition := s.step.(conditionStep)
	if isCondition {
		retryTimeout = c.timeout
	} else {
		retryTimeout = s.retryTimeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	// Run the step immediately. If an Azure authorization error is returned and
	// we have not hit the retry timeout, the authorizer is refreshed and the
	// step is called again after runner.pollInterval. If we have timed out or
	// any other error is returned, the error from the step is returned
	// directly.
	return wait.PollImmediateUntil(s.pollInterval, func() (bool, error) {
		var done bool
		var err error

		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		// Also, if it's a condition, run the inner function directly.
		if isCondition {
			done, err = c.f(ctx)
		} else {
			err = s.step.run(ctx, log)
			done = true
		}

		// Don't refresh if we have timed out
		if timeoutCtx.Err() == nil &&
			(azureerrors.HasAuthorizationFailedError(err) ||
				azureerrors.HasLinkedAuthorizationFailedError(err)) {
			log.Print(err)
			// https://github.com/Azure/ARO-RP/issues/541: it is unclear if this
			// refresh helps or not
			err = s.authorizer.RefreshWithContext(ctx)
			return false, err
		}
		return done, err
	}, timeoutCtx.Done())
}
func (s authorizationRefreshingStep) String() string {
	return fmt.Sprintf("[RetryOnAuthorizationFailedError %s]", s.step)
}
