package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

func managedIdentityCredentials(censor bool, delegatedResources []*dataplane.DelegatedResource, explicitIdentities []*dataplane.UserAssignedIdentityCredentials) dataplane.ManagedIdentityCredentials {
	return dataplane.ManagedIdentityCredentials{
		AuthenticationEndpoint: pointerutils.ToPtr("AuthenticationEndpoint"),
		CannotRenewAfter:       pointerutils.ToPtr("CannotRenewAfter"),
		ClientID:               pointerutils.ToPtr("ClientID"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return pointerutils.ToPtr("ClientSecret")
		}(),
		ClientSecretURL: pointerutils.ToPtr("ClientSecretURL"),
		CustomClaims:    pointerutils.ToPtr(customClaims()),
		DelegatedResources: func() []*dataplane.DelegatedResource {
			if len(delegatedResources) > 0 {
				return delegatedResources
			}
			return nil
		}(),
		DelegationURL: pointerutils.ToPtr("DelegationURL"),
		ExplicitIdentities: func() []*dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return explicitIdentities
			}
			return nil
		}(),
		InternalID:                 pointerutils.ToPtr("InternalID"),
		MtlsAuthenticationEndpoint: pointerutils.ToPtr("MtlsAuthenticationEndpoint"),
		NotAfter:                   pointerutils.ToPtr("NotAfter"),
		NotBefore:                  pointerutils.ToPtr("NotBefore"),
		ObjectID:                   pointerutils.ToPtr("ObjectID"),
		RenewAfter:                 pointerutils.ToPtr("RenewAfter"),
		TenantID:                   pointerutils.ToPtr("TenantID"),
	}
}

func delegatedResource(implicitIdentity *dataplane.UserAssignedIdentityCredentials, explicitIdentities ...*dataplane.UserAssignedIdentityCredentials) *dataplane.DelegatedResource {
	return &dataplane.DelegatedResource{
		DelegationID:  pointerutils.ToPtr("DelegationID"),
		DelegationURL: pointerutils.ToPtr("DelegationURL"),
		ExplicitIdentities: func() []*dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return explicitIdentities
			}
			return nil
		}(),
		ImplicitIdentity: implicitIdentity,
		InternalID:       pointerutils.ToPtr("InternalID"),
		ResourceID:       pointerutils.ToPtr("ResourceID"),
	}
}

func userAssignedIdentityCredentials(censor bool) *dataplane.UserAssignedIdentityCredentials {
	return &dataplane.UserAssignedIdentityCredentials{
		AuthenticationEndpoint: pointerutils.ToPtr("AuthenticationEndpoint"),
		CannotRenewAfter:       pointerutils.ToPtr("CannotRenewAfter"),
		ClientID:               pointerutils.ToPtr("ClientID"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return pointerutils.ToPtr("ClientSecret")
		}(),
		ClientSecretURL:            pointerutils.ToPtr("ClientSecretURL"),
		CustomClaims:               pointerutils.ToPtr(customClaims()),
		MtlsAuthenticationEndpoint: pointerutils.ToPtr("MtlsAuthenticationEndpoint"),
		NotAfter:                   pointerutils.ToPtr("NotAfter"),
		NotBefore:                  pointerutils.ToPtr("NotBefore"),
		ObjectID:                   pointerutils.ToPtr("ObjectID"),
		RenewAfter:                 pointerutils.ToPtr("RenewAfter"),
		ResourceID:                 pointerutils.ToPtr("ResourceID"),
		TenantID:                   pointerutils.ToPtr("TenantID"),
	}
}

func customClaims() dataplane.CustomClaims {
	return dataplane.CustomClaims{
		XMSAzNwperimid: []*string{pointerutils.ToPtr("XMSAzNwperimid")},
		XMSAzTm:        pointerutils.ToPtr("XMSAzTm"),
	}
}

func TestCensorCredentials(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		generateData func(censor bool) (data *dataplane.ManagedIdentityCredentials)
	}{
		{
			name: "no delegated resources, explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return pointerutils.ToPtr(managedIdentityCredentials(censor, nil, nil))
			},
		},
		{
			name: "delegated resource without explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return pointerutils.ToPtr(managedIdentityCredentials(censor, []*dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor)),
					nil,
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return pointerutils.ToPtr(managedIdentityCredentials(censor, []*dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), nil),
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return pointerutils.ToPtr(managedIdentityCredentials(censor, []*dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), nil),
				}, []*dataplane.UserAssignedIdentityCredentials{
					userAssignedIdentityCredentials(censor),
					userAssignedIdentityCredentials(censor),
					nil,
				}))
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input, output := testCase.generateData(false), testCase.generateData(true)
			CensorManagedIdentityCredentials(input)
			if diff := cmp.Diff(output, input); diff != "" {
				t.Errorf("CensorManagedIdentityCredentials mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
