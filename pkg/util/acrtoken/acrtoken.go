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
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type Manager interface {
	GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile
	NewRegistryProfile() *api.RegistryProfile
	PutRegistryProfile(oc *api.OpenShiftCluster, registryProfile *api.RegistryProfile)
	EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, error)
	RotateTokenPassword(ctx context.Context, registryProfile *api.RegistryProfile) error
	Delete(ctx context.Context, registryProfile *api.RegistryProfile) error
}

type manager struct {
	env env.Interface
	r   azure.Resource

	tokens     armcontainerregistry.TokensClient
	registries armcontainerregistry.RegistriesClient

	uuid uuid.Generator
	now  func() time.Time
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
		uuid:       uuid.DefaultGenerator,
		now:        time.Now,
	}

	return m, nil
}

func NewManagerWithClients(env env.Interface, tokensClient armcontainerregistry.TokensClient, registriesClient armcontainerregistry.RegistriesClient) (Manager, error) {
	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		return nil, err
	}

	m := &manager{
		env: env,
		r:   r,

		tokens:     tokensClient,
		registries: registriesClient,
		uuid:       uuid.DefaultGenerator,
		now:        time.Now,
	}

	return m, nil
}

func (m *manager) GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile {
	for i, registryProfile := range oc.Properties.RegistryProfiles {
		if registryProfile.Name == m.env.ACRDomain() {
			return oc.Properties.RegistryProfiles[i]
		}
	}

	return nil
}

func GetRegistryProfileFromSlice(_env env.Interface, registryProfiles []*api.RegistryProfile) *api.RegistryProfile {
	for _, registryProfile := range registryProfiles {
		if registryProfile.Name == _env.ACRDomain() {
			return registryProfile
		}
	}

	return nil
}

func (m *manager) NewRegistryProfile() *api.RegistryProfile {
	currentTime := m.now().UTC()
	return &api.RegistryProfile{
		Name:      m.env.ACRDomain(),
		Username:  "token-" + m.uuid.Generate(),
		IssueDate: &currentTime,
	}
}

func (m *manager) PutRegistryProfile(oc *api.OpenShiftCluster, registryProfile *api.RegistryProfile) {
	for i, _existingRegistryProfile := range oc.Properties.RegistryProfiles {
		if _existingRegistryProfile.Name == registryProfile.Name {
			oc.Properties.RegistryProfiles[i] = registryProfile
			return
		}
	}

	oc.Properties.RegistryProfiles = append(oc.Properties.RegistryProfiles, registryProfile)
}

// EnsureTokenAndPassword ensures a token exists with the given username,
// generates a new password for it and returns it
// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions
func (m *manager) EnsureTokenAndPassword(ctx context.Context, registryProfile *api.RegistryProfile) (string, error) {
	// We don't use anything from the token body so just ignore it
	_, err := m.tokens.CreateAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, registryProfile.Username, sdkarmcontainerregistry.Token{
		Properties: &sdkarmcontainerregistry.TokenProperties{
			ScopeMapID: pointerutils.ToPtr(m.env.ACRResourceID() + "/scopeMaps/_repositories_pull"),
			Status:     pointerutils.ToPtr(sdkarmcontainerregistry.TokenStatusEnabled),
		},
	})
	// Ignore StatusConflict errors (it means it's already created)
	if err != nil && !azureerrors.IsStatusConflictError(err) {
		return "", err
	}

	return m.generateTokenPassword(ctx, sdkarmcontainerregistry.TokenPasswordNamePassword1, registryProfile)
}

// RotateTokenPassword chooses either the unused token password or the token
// password with the oldest creation date, generates a new password, and
// then updates the registry profile with the newly generated password.
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
	// Passwords has two entries: compare creation dates, renew oldest
	case len(tokenPasswords) == 2:
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
	// default case, including passwords having zero entries: generate password 1
	// this shouldn't ever happen, which guarantees it will happen
	default:
		passwordToRenew = sdkarmcontainerregistry.TokenPasswordNamePassword1
	}

	newPassword, err := m.generateTokenPassword(ctx, passwordToRenew, registryProfile)
	if err != nil {
		return err
	}
	registryProfile.Password = api.SecureString(newPassword)
	return nil
}

// generateTokenPassword takes an existing ACR token and generates
// a password for the specified password name
func (m *manager) generateTokenPassword(ctx context.Context, passwordName sdkarmcontainerregistry.TokenPasswordName, registryProfile *api.RegistryProfile) (string, error) {
	creds, err := m.registries.GenerateCredentialsAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, sdkarmcontainerregistry.GenerateCredentialsParameters{
		TokenID: pointerutils.ToPtr(m.env.ACRResourceID() + "/tokens/" + registryProfile.Username),
		Name:    pointerutils.ToPtr(passwordName),
	})
	if err != nil {
		return "", err
	}

	// response details from Azure API
	// https://learn.microsoft.com/en-us/rest/api/containerregistry/tokens/create?tabs=Go#tokencreate

	for _, pw := range creds.Passwords {
		if pw.Name != nil && *pw.Name == passwordName {
			return *pw.Value, nil
		}
	}

	return *(creds.Passwords)[0].Value, nil
}

func (m *manager) Delete(ctx context.Context, registryProfile *api.RegistryProfile) error {
	err := m.tokens.DeleteAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, registryProfile.Username)
	// Ignore not-founds on delete
	if err != nil && azureerrors.IsNotFoundError(err) {
		return nil
	}
	return err
}
