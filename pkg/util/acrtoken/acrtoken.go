package acrtoken

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2019-12-01-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerregistry"
)

type Manager interface {
	GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile
	NewRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile
	PutRegistryProfile(oc *api.OpenShiftCluster, rp *api.RegistryProfile)
	EnsureTokenAndPassword(ctx context.Context, rp *api.RegistryProfile) (string, error)
	Delete(ctx context.Context, rp *api.RegistryProfile) error
	ApprovePrivateEndpoint(ctx context.Context, oc *api.OpenShiftCluster) error
}

type manager struct {
	env env.Interface
	r   azure.Resource

	tokens     containerregistry.TokensClient
	registries containerregistry.RegistriesClient
	pec        containerregistry.PrivateEndpointConnectionsClient
}

func NewManager(env env.Interface, localFPAuthorizer autorest.Authorizer) (Manager, error) {
	r, err := azure.ParseResourceID(env.ACRResourceID())
	if err != nil {
		return nil, err
	}

	m := &manager{
		env: env,
		r:   r,

		tokens:     containerregistry.NewTokensClient(r.SubscriptionID, localFPAuthorizer),
		registries: containerregistry.NewRegistriesClient(r.SubscriptionID, localFPAuthorizer),
		pec:        containerregistry.NewPrivateEndpointConnectionsClient(r.SubscriptionID, localFPAuthorizer),
	}

	return m, nil
}

func (m *manager) GetRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile {
	for i, rp := range oc.Properties.RegistryProfiles {
		if rp.Name == m.r.ResourceName+".azurecr.io" {
			return oc.Properties.RegistryProfiles[i]
		}
	}

	return nil
}

func (m *manager) NewRegistryProfile(oc *api.OpenShiftCluster) *api.RegistryProfile {
	return &api.RegistryProfile{
		Name:     m.r.ResourceName + ".azurecr.io",
		Username: "token-" + uuid.NewV4().String(),
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

	creds, err := m.registries.GenerateCredentials(ctx, m.r.ResourceGroup, m.r.ResourceName, mgmtcontainerregistry.GenerateCredentialsParameters{
		TokenID: to.StringPtr(m.env.ACRResourceID() + "/tokens/" + rp.Username),
		Name:    mgmtcontainerregistry.TokenPasswordNamePassword1,
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

func (m *manager) ApprovePrivateEndpoint(ctx context.Context, oc *api.OpenShiftCluster) error {
	infraID := oc.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}
	peName := infraID + "-arosvc-pe"
	// Private endpoint name is not primary input for pec.Get. We have to list :/
	pecs, err := m.pec.List(ctx, m.r.ResourceGroup, m.r.ResourceName)
	if err != nil {
		return err
	}

	for _, pec := range pecs {
		r, err := azure.ParseResourceID(*pec.PrivateEndpointConnectionProperties.PrivateEndpoint.ID)
		if err != nil {
			return err
		}
		if r.ResourceName == peName &&
			pec.PrivateEndpointConnectionProperties.PrivateLinkServiceConnectionState.Status != mgmtcontainerregistry.Approved {
			pec.PrivateEndpointConnectionProperties.PrivateLinkServiceConnectionState.Status = mgmtcontainerregistry.Approved
			_, err := m.pec.CreateOrUpdate(ctx, m.r.ResourceGroup, m.r.ResourceName, *pec.Name, pec)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
