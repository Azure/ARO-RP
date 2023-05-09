package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

var ErrWantRefresh = errors.New("want refresh")

// AuthorizationRefreshingAction returns a wrapper Step which will refresh
// `authorizer` if the step returns an Azure AuthenticationError and rerun it.
// The step will be retried until `retryTimeout` is hit. Any other error will be
// returned directly.
func AuthorizationRetryingAction(r refreshable.Authorizer, action actionFunction) Step {
	return &authorizationRefreshingActionStep{
		auth: r,
		f:    action,
	}
}

type authorizationRefreshingActionStep struct {
	f            actionFunction
	auth         refreshable.Authorizer
	retryTimeout time.Duration
	pollInterval time.Duration
}

func (s *authorizationRefreshingActionStep) run(ctx context.Context, log *logrus.Entry) error {
	var pollInterval time.Duration
	var retryTimeout time.Duration

	// ARM role caching can be 5 minutes
	if s.retryTimeout == time.Duration(0) {
		retryTimeout = 10 * time.Minute
	} else {
		retryTimeout = s.retryTimeout
	}

	// If no pollInterval has been set, use a default
	if s.pollInterval == time.Duration(0) {
		pollInterval = 30 * time.Second
	} else {
		pollInterval = s.pollInterval
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	// Run the step immediately. If an Azure authorization error is returned and
	// we have not hit the retry timeout, the authorizer is refreshed and the
	// step is called again after runner.pollInterval. If we have timed out or
	// any other error is returned, the error from the step is returned
	// directly.
	return wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		err := s.f(ctx)

		// If we haven't timed out and there is an error that is either an
		// unauthorized client (AADSTS700016) or "AuthorizationFailed" (likely
		// role propagation delay) then refresh and retry.
		if timeoutCtx.Err() == nil && err != nil &&
			(azureerrors.IsUnauthorizedClientError(err) ||
				azureerrors.HasAuthorizationFailedError(err) ||
				azureerrors.IsInvalidSecretError(err) ||
				err == ErrWantRefresh) {
			if s.auth != nil {
				log.Printf("auth error, refreshing and retrying: %v", err)
				// Try refreshing auth.
				err = s.auth.Rebuild()
			} else {
				log.Printf("auth error, retrying: %v", err)
			}
			return false, err // retry step
		}
		return true, err
	}, timeoutCtx.Done())
}

func (s *authorizationRefreshingActionStep) String() string {
	return fmt.Sprintf("[AuthorizationRetryingAction %s]", FriendlyName(s.f))
}

func (s *authorizationRefreshingActionStep) metricsName() string {
	return fmt.Sprintf("authorizationretryingaction.%s", shortName(FriendlyName(s.f)))
}
