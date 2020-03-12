package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	azcontainerregistry "github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-06-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerregistry"
)

type Manager interface {
	GetTokenName(oc *api.OpenShiftCluster) string
	CreateToken(ctx context.Context, tokenName string) error
	CreatePassword(ctx context.Context, oc *api.OpenShiftCluster) (string, error)
	SetRegistryProfileUsername(oc *api.OpenShiftCluster, username string) error
	SetRegistryProfilePassword(oc *api.OpenShiftCluster, password string) error
	Delete(ctx context.Context, oc *api.OpenShiftCluster) error
}

type manager struct {
	env env.Interface

	tokens     containerregistry.TokensClient
	registries containerregistry.RegistriesClient

	scopeMap string
}

func NewManager(env env.Interface, fpAuthorizer autorest.Authorizer) Manager {
	return &manager{
		env: env,

		tokens:     containerregistry.NewTokensClient(env.SubscriptionID(), fpAuthorizer),
		registries: containerregistry.NewRegistriesClient(env.SubscriptionID(), fpAuthorizer),

		scopeMap: fmt.Sprintf("%s/scopeMaps/_repositories_pull", env.ACRResourceID()),
	}
}

// GetRegistryProfile get the registry profile of the type provided
func (m *manager) getRegistryProfile(oc *api.OpenShiftCluster, rtype api.RegistryType) *api.RegistryProfile {
	for ix, rp := range oc.Properties.RegistryProfiles {
		if rp.Type == rtype {
			return &oc.Properties.RegistryProfiles[ix]
		}
	}
	return nil
}

func (m *manager) SetRegistryProfileUsername(oc *api.OpenShiftCluster, username string) error {
	acrRP := m.getRegistryProfile(oc, api.RegistryTypeACR)
	if acrRP == nil {
		r, err := azure.ParseResourceID(m.env.ACRResourceID())
		if err != nil {
			return err
		}
		oc.Properties.RegistryProfiles = append(oc.Properties.RegistryProfiles, api.RegistryProfile{
			Type: api.RegistryTypeACR,
			Name: r.ResourceName + ".azurecr.io",
		})
		acrRP = m.getRegistryProfile(oc, api.RegistryTypeACR)
	}
	acrRP.Username = username
	return nil
}

func (m *manager) SetRegistryProfilePassword(oc *api.OpenShiftCluster, password string) error {
	acrRP := m.getRegistryProfile(oc, api.RegistryTypeACR)
	if acrRP == nil {
		return fmt.Errorf("registryProfile %s not found", api.RegistryTypeACR)
	}
	acrRP.Password = api.SecureString(password)
	return nil
}

func (m *manager) GetTokenName(oc *api.OpenShiftCluster) string {
	if m.env.ACRResourceID() == "" { // currently only dev will not have per cluster ACR tokens
		return ""
	}
	acrRP := m.getRegistryProfile(oc, api.RegistryTypeACR)
	if acrRP != nil && acrRP.Username != "" {
		return acrRP.Username
	}
	return "token-" + uuid.NewV4().String()
}

// Create create a token on our registry
// see https://docs.microsoft.com/en-us/azure/container-registry/container-registry-repository-scoped-permissions
func (m *manager) CreateToken(ctx context.Context, tokenName string) error {
	if m.env.ACRResourceID() == "" || tokenName == "" { // currently only dev will not have per cluster ACR tokens
		return nil
	}

	r, err := azure.ParseResourceID(m.env.ACRResourceID())
	if err != nil {
		return err
	}
	err = m.tokens.CreateAndWait(ctx, r.ResourceGroup, r.ResourceName, tokenName, azcontainerregistry.Token{
		TokenProperties: &azcontainerregistry.TokenProperties{
			ScopeMapID: &m.scopeMap,
			Status:     azcontainerregistry.TokenStatusEnabled,
		},
	})
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok && (detailedErr.StatusCode == http.StatusConflict) {
			return nil
		}
		return err
	}
	return nil
}

func (m *manager) CreatePassword(ctx context.Context, oc *api.OpenShiftCluster) (string, error) {
	if m.env.ACRResourceID() == "" { // currently only dev will not have per cluster ACR tokens
		return "", nil
	}
	acrRP := m.getRegistryProfile(oc, api.RegistryTypeACR)
	if acrRP == nil {
		return "", fmt.Errorf("RegistryProfile not found")
	}
	if acrRP.Password != "" {
		return string(acrRP.Password), nil
	}
	r, err := azure.ParseResourceID(m.env.ACRResourceID())
	if err != nil {
		return "", err
	}

	generateCredentialsParameters := azcontainerregistry.GenerateCredentialsParameters{
		TokenID: to.StringPtr(fmt.Sprintf("%s/tokens/%s", m.env.ACRResourceID(), acrRP.Username)),
		Name:    azcontainerregistry.TokenPasswordNamePassword1,
	}
	creds, err := m.registries.GenerateCredentials(ctx, r.ResourceGroup, r.ResourceName, generateCredentialsParameters)
	if err != nil {
		return "", err
	}
	if creds.Passwords == nil || len(*creds.Passwords) < 1 || (*creds.Passwords)[0].Value == nil {
		return "", fmt.Errorf("generateCredentials returned empty passwords")
	}
	return *(*creds.Passwords)[0].Value, nil
}

func (m *manager) Delete(ctx context.Context, oc *api.OpenShiftCluster) error {
	if m.env.ACRResourceID() == "" { // currently only dev will not have per cluster ACR tokens
		return nil
	}
	r, err := azure.ParseResourceID(m.env.ACRResourceID())
	if err != nil {
		return err
	}

	for _, rp := range oc.Properties.RegistryProfiles {
		if rp.Type == api.RegistryTypeACR && rp.Username != "" {
			err := m.tokens.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, rp.Username)
			if err != nil {
				if detailedErr, ok := err.(autorest.DetailedError); ok && detailedErr.StatusCode == http.StatusNotFound {
					return nil
				}
				return err
			}
		}
	}
	return nil
}
