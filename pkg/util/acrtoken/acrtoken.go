package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerregistry"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type Manager interface {
	GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile
	NewRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile
	PutRegistryProfile(oc *api.OpenShiftCluster, rp *api.RegistryProfile)
	EnsureTokenAndPassword(ctx context.Context, rp *api.RegistryProfile) (string, error)
	RotateTokenPassword(ctx context.Context, rp *api.RegistryProfile) error
	Delete(ctx context.Context, rp *api.RegistryProfile) error
}

type manager struct {
	env env.Interface
	r   azure.Resource

	tokens     containerregistry.TokensClient
	registries containerregistry.RegistriesClient
}

func NewManager(env env.Interface, localFPAuthorizer autorest.Authorizer) (Manager, error) {
	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		return nil, err
	}

	m := &manager{
		env: env,
		r:   r,

		tokens:     containerregistry.NewTokensClient(env.Environment(), r.SubscriptionID, localFPAuthorizer),
		registries: containerregistry.NewRegistriesClient(env.Environment(), r.SubscriptionID, localFPAuthorizer),
	}

	return m, nil
}

func (m *manager) GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile {
	for i, rp := range oc.Properties.RegistryProfiles {
		if rp.Name == fmt.Sprintf("%s.%s", m.r.ResourceName, m.env.Environment().ContainerRegistryDNSSuffix) {
			return oc.Properties.RegistryProfiles[i]
		}
	}

	return nil
}

func (m *manager) NewRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile {
	return &api.RegistryProfile{
		Name:     fmt.Sprintf("%s.%s", m.r.ResourceName, m.env.Environment().ContainerRegistryDNSSuffix),
		Username: "token-" + uuid.DefaultGenerator.Generate(),
	}
}

func (m *manager) PutRegistryProfile(oc *api.OpenShiftCluster, rp *api.RegistryProfile) {
	for i, _rp := range oc.Properties.RegistryProfiles {
		if _rp.Name == rp.Name {
			oc.Properties.RegistryProfiles[i] = rp
			return
		}
	}

	oc.Properties.RegistryProfiles = append(oc.Properties.RegistryProfiles, rp)
}

// EnsureTokenAndPassword ensures a token exists with the given username,
// generates a new password for it and returns it
// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions
func (m *manager) EnsureTokenAndPassword(ctx context.Context, rp *api.RegistryProfile) (string, error) {
	err := m.tokens.CreateAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, rp.Username, mgmtcontainerregistry.Token{
		TokenProperties: &mgmtcontainerregistry.TokenProperties{
			ScopeMapID: to.StringPtr(m.env.ACRResourceID() + "/scopeMaps/_repositories_pull"),
			Status:     mgmtcontainerregistry.TokenStatusEnabled,
		},
	})
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusConflict {
		err = nil
	}
	if err != nil {
		return "", err
	}

	return m.generateTokenPassword(ctx, mgmtcontainerregistry.TokenPasswordNamePassword1, rp)
}

// RotateTokenPassword chooses either the unused token password or the token
// password with the oldest creation date, generates a new password, and
// then updates the registry profile with the newly generated password.
func (m *manager) RotateTokenPassword(ctx context.Context, rp *api.RegistryProfile) error {
	tokenProperties, err := m.tokens.GetTokenProperties(ctx, m.r.ResourceGroup, m.r.ResourceName, rp.Username)
	if err != nil {
		return err
	}

	// in practice there are only 2 possible passwords for the token, but the slice as written has arbitrary length so we iterate over all of them
	var toRenew mgmtcontainerregistry.TokenPassword
	for _, tp := range *tokenProperties.Credentials.Passwords {
		if tp.CreationTime.IsZero() {
			toRenew = tp
		} else if tp.CreationTime.Before(toRenew.CreationTime.ToTime()) {
			toRenew = tp
		}
	}

	newPassword, err := m.generateTokenPassword(ctx, toRenew.Name, rp)
	if err != nil {
		return err
	}
	rp.Password = api.SecureString(newPassword)
	return nil
}

// generateTokenPassword takes an existing ACR token and generates
// a password for the specified password name
func (m *manager) generateTokenPassword(ctx context.Context, passwordName mgmtcontainerregistry.TokenPasswordName, rp *api.RegistryProfile) (string, error) {

	creds, err := m.registries.GenerateCredentials(ctx, m.r.ResourceGroup, m.r.ResourceName, mgmtcontainerregistry.GenerateCredentialsParameters{
		TokenID: to.StringPtr(m.env.ACRResourceID() + "/tokens/" + rp.Username),
		Name:    passwordName,
	})
	if err != nil {
		return "", err
	}

	return *(*creds.Passwords)[0].Value, nil
}

func (m *manager) Delete(ctx context.Context, rp *api.RegistryProfile) error {
	err := m.tokens.DeleteAndWait(ctx, m.r.ResourceGroup, m.r.ResourceName, rp.Username)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		err = nil
	}
	return err
}
