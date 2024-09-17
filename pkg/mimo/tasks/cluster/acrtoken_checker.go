package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/mimo"
)

const (
	daysValid        = 90
	daysShouldRotate = 45
)

// EnsureACRTokenIsValid checks the expiry date of the Azure Container Registry (ACR) Token from the RegistryProfile.
// It returns an error if the expiry date is past the date now or if there is no registry profile found.
func EnsureACRTokenIsValid(ctx context.Context) error {
	th, err := mimo.GetTaskContext(ctx)
	if err != nil {
		return mimo.TerminalError(err)
	}

	env := th.Environment()
	localFpAuthorizer, err := th.LocalFpAuthorizer()
	if err != nil {
		return mimo.TerminalError(err)
	}

	manager, err := acrtoken.NewManager(env, localFpAuthorizer)
	if err != nil {
		return err
	}

	registryProfiles := th.GetOpenShiftClusterProperties().RegistryProfiles
	rp := manager.GetRegistryProfileFromSlice(registryProfiles)
	if rp != nil {
		var now = time.Now().UTC()
		issueDate := rp.IssueDate

		if issueDate == nil {
			return mimo.TerminalError(errors.New("no expiry date detected"))
		}

		daysInterval := int32(now.Sub(issueDate.Time).Hours() / 24)

		switch {
		case daysInterval > daysValid:
			return mimo.TerminalError(fmt.Errorf("azure container registry (acr) token has expired, %d days have passed", daysInterval))
		case daysInterval >= daysShouldRotate:
			return mimo.TerminalError(fmt.Errorf("%d days have passed since azure container registry (acr) token was issued, please rotate the token now", daysInterval))
		default:
			th.SetResultMessage("azure container registry (acr) token is valid")
		}
	} else {
		return mimo.TerminalError(errors.New("no registry profile detected"))
	}

	return nil
}
