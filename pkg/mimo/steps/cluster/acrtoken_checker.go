package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

// EnsureACRTokenIsValid checks the expiry date of the Azure Container Registry (ACR) Token from the RegistryProfile.
// It returns an error if the expiry date is past the date now or if there is no registry profile found.
func EnsureACRTokenIsValid(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	env := th.Environment()
	registryProfiles := th.GetOpenShiftClusterProperties().RegistryProfiles
	rp := acrtoken.GetRegistryProfileFromSlice(env, registryProfiles)

	if rp == nil || rp.IssueDate == nil {
		return mimo.TerminalError(errors.New("no issue date detected, please rotate token"))
	}

	shouldRotate, isValid, timeUntilNextRotate, validityRemaining := acrtoken.ShouldRotateToken(env, rp)

	if !isValid {
		return mimo.TerminalError(errors.New("token is expired"))
	} else if shouldRotate {
		return mimo.TerminalError(fmt.Errorf("%s since ACR token should be rotated, %s validity remaining, please rotate", timeUntilNextRotate.String(), validityRemaining.String()))
	}
	th.SetResultMessage(fmt.Sprintf("token validity has %s remaining, should be rotated in %s", validityRemaining.String(), timeUntilNextRotate.String()))
	return nil
}
