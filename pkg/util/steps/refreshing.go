package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
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
	var (
		err          error
		pollInterval time.Duration
		retryTimeout time.Duration
	)

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

	// Propagate the latest authorization error to the user,
	// rather than timeout error from PollImmediateUntil.
	_ = wait.PollImmediateUntil(pollInterval, func() (bool, error) {
		// We use the outer context, not the timeout context, as we do not want
		// to time out the condition function itself, only stop retrying once
		// timeoutCtx's timeout has fired.
		err = s.f(ctx)

		// If we haven't timed out and there is an error that is either an
		// unauthorized client (AADSTS700016) or "AuthorizationFailed" (likely
		// role propagation delay) then refresh and retry.
		if timeoutCtx.Err() == nil && err != nil &&
			(azureerrors.IsUnauthorizedClientError(err) ||
				azureerrors.HasAuthorizationFailedError(err) ||
				azureerrors.IsInvalidSecretError(err) ||
				azureerrors.IsDeploymentMissingPermissionsError(err) ||
				err == ErrWantRefresh) {
			log.Printf("auth error, refreshing and retrying: %v", err)
			// Try refreshing auth.
			err = s.auth.Rebuild()
			return false, err // retry step
		}
		if err != nil {
			log.Printf("non-auth error, giving up: %v", err)
		}
		return true, err
	}, timeoutCtx.Done())

	// After timeout, return any actionable errors to the user
	if err != nil {
		switch {
		case azureerrors.IsUnauthorizedClientError(err):
			return s.servicePrincipalCloudError(
				"The provided service principal application (client) ID was not found in the directory (tenant). Please ensure that the provided application (client) id and client secret value are correct.",
			)
		case azureerrors.HasAuthorizationFailedError(err) || azureerrors.IsInvalidSecretError(err):
			return s.servicePrincipalCloudError(
				"Authorization using provided credentials failed. Please ensure that the provided application (client) id and client secret value are correct.",
			)
		default:
			// If not actionable, still log err in RP logs
			return err
		}
	}

	return nil
}

func (s *authorizationRefreshingActionStep) String() string {
	return fmt.Sprintf("[AuthorizationRetryingAction %s]", FriendlyName(s.f))
}

func (s *authorizationRefreshingActionStep) metricsName() string {
	return fmt.Sprintf("authorizationretryingaction.%s", shortName(FriendlyName(s.f)))
}

func (s *authorizationRefreshingActionStep) servicePrincipalCloudError(message string) error {
	return api.NewCloudError(
		http.StatusBadRequest,
		api.CloudErrorCodeInvalidServicePrincipalCredentials,
		"properties.servicePrincipalProfile",
		message)
}
