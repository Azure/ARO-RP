package steps

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	_ = wait.PollUntilContextCancel(timeoutCtx, pollInterval, true, func(ctx context.Context) (bool, error) {
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
	})

	return CreateActionableError(err)
}

func (s *authorizationRefreshingActionStep) String() string {
	return fmt.Sprintf("[AuthorizationRetryingAction %s]", FriendlyName(s.f))
}

func (s *authorizationRefreshingActionStep) metricsName() string {
	return fmt.Sprintf("authorizationretryingaction.%s", shortName(FriendlyName(s.f)))
}

// Creates a one line string from a series of strings delimited by a space character.
func make_one_line_str(tokens ...string) string {
	return strings.Join(tokens, " ")
}

// Creates a CloudError due to an invalid service principal error.
func newServicePrincipalCloudError(message string, statusCode int) error {
	return api.NewCloudError(
		statusCode,
		api.CloudErrorCodeInvalidServicePrincipalCredentials,
		"properties.servicePrincipalProfile",
		message)
}

// Creates a user-actionable error from an error.
// NOTE: the resultant error must be user-friendly
// as this may be potentially be presented to a user.
func CreateActionableError(err error) error {
	// Log this error for debugging new error types.
	log.Printf("Converting to user actionable error: %v [%T]", err, err)

	if err == nil {
		return err
	}

	switch {
	case azureerrors.IsUnauthorizedClientError(err):
		return newServicePrincipalCloudError(make_one_line_str(
			"The provided service principal application",
			"(client) ID was not found in the directory",
			"(tenant). Please ensure that the provided",
			"application (client) id and client secret",
			"value are correct."),
			http.StatusBadRequest,
		)
	case azureerrors.HasAuthorizationFailedError(err) || azureerrors.IsInvalidSecretError(err):
		return newServicePrincipalCloudError(make_one_line_str(
			"Authorization using provided credentials failed.",
			"Please ensure that the provided application (client)",
			"id and client secret value are correct."),
			http.StatusBadRequest,
		)
	case azureerrors.IsClientSecretKeysExpired(err):
		return newServicePrincipalCloudError(make_one_line_str(
			"The provided client secret is expired.",
			"Please create a new one for your service principal."),
			http.StatusBadRequest,
		)
	default:
		log.Printf("Unable to convert to actionable error: %v", err)
		return err
	}
}
