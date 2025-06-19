package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
)

func managedIdentityCredentials(censor bool, delegatedResources []dataplane.DelegatedResource, explicitIdentities []dataplane.UserAssignedIdentityCredentials) dataplane.ManagedIdentityCredentials {
	return dataplane.ManagedIdentityCredentials{
		AuthenticationEndpoint: to.Ptr("AuthenticationEndpoint"),
		CannotRenewAfter:       to.Ptr("CannotRenewAfter"),
		ClientID:               to.Ptr("ClientID"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return to.Ptr("ClientSecret")
		}(),
		ClientSecretURL: to.Ptr("ClientSecretURL"),
		CustomClaims:    to.Ptr(customClaims()),
		DelegatedResources: func() []dataplane.DelegatedResource {
			if len(delegatedResources) > 0 {
				return delegatedResources
			}
			return nil
		}(),
		DelegationURL: to.Ptr("DelegationURL"),
		ExplicitIdentities: func() []dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return explicitIdentities
			}
			return nil
		}(),
		InternalID:                 to.Ptr("InternalID"),
		MtlsAuthenticationEndpoint: to.Ptr("MtlsAuthenticationEndpoint"),
		NotAfter:                   to.Ptr("NotAfter"),
		NotBefore:                  to.Ptr("NotBefore"),
		ObjectID:                   to.Ptr("ObjectID"),
		RenewAfter:                 to.Ptr("RenewAfter"),
		TenantID:                   to.Ptr("TenantID"),
	}
}

func delegatedResource(implicitIdentity dataplane.UserAssignedIdentityCredentials, explicitIdentities ...dataplane.UserAssignedIdentityCredentials) dataplane.DelegatedResource {
	return dataplane.DelegatedResource{
		DelegationID:  to.Ptr("DelegationID"),
		DelegationURL: to.Ptr("DelegationURL"),
		ExplicitIdentities: func() []dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return explicitIdentities
			}
			return nil
		}(),
		ImplicitIdentity: to.Ptr(implicitIdentity),
		InternalID:       to.Ptr("InternalID"),
		ResourceID:       to.Ptr("ResourceID"),
	}
}

func userAssignedIdentityCredentials(censor bool) dataplane.UserAssignedIdentityCredentials {
	return dataplane.UserAssignedIdentityCredentials{
		AuthenticationEndpoint: to.Ptr("AuthenticationEndpoint"),
		CannotRenewAfter:       to.Ptr("CannotRenewAfter"),
		ClientID:               to.Ptr("ClientID"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return to.Ptr("ClientSecret")
		}(),
		ClientSecretURL:            to.Ptr("ClientSecretURL"),
		CustomClaims:               to.Ptr(customClaims()),
		MtlsAuthenticationEndpoint: to.Ptr("MtlsAuthenticationEndpoint"),
		NotAfter:                   to.Ptr("NotAfter"),
		NotBefore:                  to.Ptr("NotBefore"),
		ObjectID:                   to.Ptr("ObjectID"),
		RenewAfter:                 to.Ptr("RenewAfter"),
		ResourceID:                 to.Ptr("ResourceID"),
		TenantID:                   to.Ptr("TenantID"),
	}
}

func customClaims() dataplane.CustomClaims {
	return dataplane.CustomClaims{
		XMSAzNwperimid: []string{"XMSAzNwperimid"},
		XMSAzTm:        to.Ptr("XMSAzTm"),
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
				return to.Ptr(managedIdentityCredentials(censor, nil, nil))
			},
		},
		{
			name: "delegated resource without explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return to.Ptr(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor)),
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return to.Ptr(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return to.Ptr(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
				}, []dataplane.UserAssignedIdentityCredentials{
					userAssignedIdentityCredentials(censor),
					userAssignedIdentityCredentials(censor),
				}))
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input, output := testCase.generateData(false), testCase.generateData(true)
			censorCredentials(input)
			if diff := cmp.Diff(output, input); diff != "" {
				t.Errorf("censorCredentials mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
