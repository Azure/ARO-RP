package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	sdkarmcontainerregistry "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

// Maximum lifetime of the ACR token
const (
	ACRTokenRotateAfterDays = 140
	ACRTokenMaxLifetimeDays = 180
	ACRTokenRotateAfter     = time.Hour * 24 * time.Duration(ACRTokenRotateAfterDays)
	ACRTokenMaxLifetime     = time.Hour * 24 * time.Duration(ACRTokenMaxLifetimeDays)
)

type Manager interface {
	NewRegistryProfile(clusterUUID string) *api.RegistryProfile
	EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, *time.Time, error)
	RotateTokenPassword(ctx context.Context, registryProfile *api.RegistryProfile) error
	Delete(ctx context.Context, registryProfile *api.RegistryProfile) error
}

type manager struct {
	env env.Interface
	r   azure.Resource

	tokens     armcontainerregistry.TokensClient
	registries armcontainerregistry.RegistriesClient
}

func NewManager(env env.Interface, tokensClient armcontainerregistry.TokensClient, registriesClient armcontainerregistry.RegistriesClient) (Manager, error) {
	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		return nil, err
	}

	m := &manager{
		env: env,
		r:   r,

		tokens:     tokensClient,
		registries: registriesClient,
	}

	return m, nil
}

func (m *manager) NewRegistryProfile(clusterUUID string) *api.RegistryProfile {
	return &api.RegistryProfile{
		Name:     m.env.ACRDomain(),
		Username: "token-" + clusterUUID,
	}
}

// EnsureTokenAndPassword ensures a token exists with the given username,
// generates a new password for it and returns it
// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions
func (m *manager) EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, *time.Time, error) {
	// We don't use anything from the token body so just ignore it
	_, err := m.tokens.CreateAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, registryProfile.Username, sdkarmcontainerregistry.Token{
		Properties: &sdkarmcontainerregistry.TokenProperties{
			ScopeMapID: pointerutils.ToPtr(m.env.ACRResourceID() + "/scopeMaps/_repositories_pull"),
			Status:     pointerutils.ToPtr(sdkarmcontainerregistry.TokenStatusEnabled),
		},
	})
	// Ignore StatusConflict errors (it means it's already created)
	if err != nil && !azureerrors.IsStatusConflictError(err) {
		return "", nil, err
	}

	return m.generateTokenPassword(ctx, sdkarmcontainerregistry.TokenPasswordNamePassword1, registryProfile)
}

// RotateTokenPassword chooses the token that is not presently in use, generates
// a new password, and then updates the registry profile with the newly
// generated password.
func (m *manager) RotateTokenPassword(ctx context.Context, registryProfile *api.RegistryProfile) error {
	tokenProperties, err := m.tokens.GetTokenProperties(ctx, m.r.ResourceGroup, m.r.ResourceName, registryProfile.Username)
	if err != nil {
		return err
	}

	var tokenPasswords []*sdkarmcontainerregistry.TokenPassword
	if tokenProperties.Credentials != nil {
		tokenPasswords = tokenProperties.Credentials.Passwords
	}

	for i, p := range tokenPasswords {
		if p.Name == nil {
			return fmt.Errorf("token password %d did not have a name (should be password1 or password2)", i)
		}
	}

	var passwordToRenew sdkarmcontainerregistry.TokenPasswordName
	switch {
	// Passwords only has one entry: renew password that isn't present
	case len(tokenPasswords) == 1:
		if *tokenPasswords[0].Name == sdkarmcontainerregistry.TokenPasswordNamePassword1 {
			passwordToRenew = sdkarmcontainerregistry.TokenPasswordNamePassword2
		} else {
			passwordToRenew = sdkarmcontainerregistry.TokenPasswordNamePassword1
		}
	// Passwords has two entries: renew password that isn't in active use by the
	// RegistryProfile (by matching issueDate), and if neither is, fall back to
	// renewing the oldest. We hope to avoid renewing the in-use password
	// because two successive rotations of the oldest that don't get applied to
	// the cluster (e.g. because timeouts) will cause the in-cluster secret to
	// become invalid and image pulls to stop working.
	case len(tokenPasswords) == 2:
		if tokenPasswords[0].CreationTime != nil &&
			registryProfile.IssueDate != nil &&
			tokenPasswords[0].CreationTime.Equal(*registryProfile.IssueDate) {
			passwordToRenew = *tokenPasswords[1].Name
		} else if tokenPasswords[1].CreationTime != nil &&
			registryProfile.IssueDate != nil &&
			tokenPasswords[1].CreationTime.Equal(*registryProfile.IssueDate) {
			passwordToRenew = *tokenPasswords[0].Name
		} else {
			var oldest *sdkarmcontainerregistry.TokenPassword
			for _, p := range tokenPasswords {
				if p.CreationTime == nil {
					oldest = p
					break
				}
			}
			if oldest == nil {
				if tokenPasswords[0].CreationTime.Before(*tokenPasswords[1].CreationTime) {
					oldest = tokenPasswords[0]
				} else {
					oldest = tokenPasswords[1]
				}
			}
			passwordToRenew = *oldest.Name
		}
	// default case, including passwords having zero entries: generate password 1
	// this shouldn't ever happen, which guarantees it will happen
	default:
		passwordToRenew = sdkarmcontainerregistry.TokenPasswordNamePassword1
	}

	newPassword, issueDate, err := m.generateTokenPassword(ctx, passwordToRenew, registryProfile)
	if err != nil {
		return err
	}
	registryProfile.Password = api.SecureString(newPassword)
	registryProfile.IssueDate = issueDate
	return nil
}

// generateTokenPassword takes an existing ACR token and generates
// a password for the specified password name
func (m *manager) generateTokenPassword(ctx context.Context, passwordName sdkarmcontainerregistry.TokenPasswordName, registryProfile *api.RegistryProfile) (string, *time.Time, error) {
	creds, err := m.registries.GenerateCredentialsAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, sdkarmcontainerregistry.GenerateCredentialsParameters{
		TokenID: pointerutils.ToPtr(m.env.ACRResourceID() + "/tokens/" + registryProfile.Username),
		Name:    pointerutils.ToPtr(passwordName),
	})
	if err != nil {
		return "", nil, err
	}

	// response details from Azure API
	// https://learn.microsoft.com/en-us/rest/api/containerregistry/tokens/create?tabs=Go#tokencreate

	for _, pw := range creds.Passwords {
		if pw.Name != nil && *pw.Name == passwordName {
			return *pw.Value, pointerutils.ToPtr(pw.CreationTime.UTC()), nil
		}
	}

	return *(creds.Passwords)[0].Value, pointerutils.ToPtr(creds.Passwords[0].CreationTime.UTC()), nil
}

func (m *manager) Delete(ctx context.Context, registryProfile *api.RegistryProfile) error {
	err := m.tokens.DeleteAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, registryProfile.Username)
	// Ignore not-founds on delete
	if err != nil && azureerrors.IsStatusNotFoundError(err) {
		return nil
	}
	return err
}

// ShouldRotateToken returns whether a token should be rotated, is before its
// validity expiry, and how long until it should be rotated and expiry will
// happen.
func ShouldRotateToken(_env env.Core, registryProfile *api.RegistryProfile) (shouldRotate bool, isValid bool, timeUntilNextRotate time.Duration, timeUntilTokenExpiry time.Duration) {
	if registryProfile == nil || registryProfile.IssueDate == nil {
		return true, false, 0, 0
	}
	now := _env.Now()
	rotateIfAfter := registryProfile.IssueDate.Add(ACRTokenRotateAfter)
	validityEnd := registryProfile.IssueDate.Add(ACRTokenMaxLifetime)

	shouldRotate = now.After(rotateIfAfter)
	isValid = validityEnd.After(now)
	timeUntilNextRotate = rotateIfAfter.Sub(now)
	if isValid {
		timeUntilTokenExpiry = validityEnd.Sub(now)
	}
	return
}
