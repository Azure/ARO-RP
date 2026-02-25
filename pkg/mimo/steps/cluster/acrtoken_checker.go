package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
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

	// TODO: Move this into the TaskContext
	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		return err
	}

	fpCredential, err := env.FPNewClientCertificateCredential(env.TenantID(), nil)
	if err != nil {
		return mimo.TerminalError(err)
	}

	tokensClient, err := armcontainerregistry.NewTokensClient(r.SubscriptionID, fpCredential, env.Environment().ArmClientOptions())
	if err != nil {
		return mimo.TerminalError(err)
	}

	registriesClient, err := armcontainerregistry.NewRegistriesClient(r.SubscriptionID, fpCredential, env.Environment().ArmClientOptions())
	if err != nil {
		return mimo.TerminalError(err)
	}

	manager, err := acrtoken.NewManager(env, tokensClient, registriesClient)
	if err != nil {
		return err
	}

	registryProfiles := th.GetOpenShiftClusterProperties().RegistryProfiles
	rp := manager.GetRegistryProfileFromSlice(registryProfiles)
	if rp != nil {
		now := time.Now().UTC()
		issueDate := rp.IssueDate

		if issueDate == nil {
			return mimo.TerminalError(errors.New("no issue date detected, please rotate token"))
		}

		daysInterval := int32(now.Sub(*issueDate).Hours() / 24)

		switch {
		case daysInterval > daysValid:
			return mimo.TerminalError(fmt.Errorf("azure container registry (acr) token is not valid, %d days have passed", daysInterval))
		case daysInterval >= daysShouldRotate:
			return mimo.TerminalError(fmt.Errorf("%d days have passed since azure container registry (acr) token was issued, please rotate the token now", daysInterval))
		default:
			th.SetResultMessage("azure container registry (acr) token is valid")
		}
	}

	return mimo.TerminalError(errors.New("no registry profile detected"))
}
