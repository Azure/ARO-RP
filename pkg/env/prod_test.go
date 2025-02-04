package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"

	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/google/go-cmp/cmp"
	"k8s.io/utils/ptr"
)

func managedIdentityCredentials(censor bool, delegatedResources []dataplane.DelegatedResource, explicitIdentities []dataplane.UserAssignedIdentityCredentials) dataplane.ManagedIdentityCredentials {
	return dataplane.ManagedIdentityCredentials{
		AuthenticationEndpoint: ptr.To("AuthenticationEndpoint"),
		CannotRenewAfter:       ptr.To("CannotRenewAfter"),
		ClientId:               ptr.To("ClientId"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return ptr.To("ClientSecret")
		}(),
		ClientSecretUrl: ptr.To("ClientSecretUrl"),
		CustomClaims:    ptr.To(customClaims()),
		DelegatedResources: func() *[]dataplane.DelegatedResource {
			if len(delegatedResources) > 0 {
				return &delegatedResources
			}
			return nil
		}(),
		DelegationUrl: ptr.To("DelegationUrl"),
		ExplicitIdentities: func() *[]dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return &explicitIdentities
			}
			return nil
		}(),
		InternalId:                 ptr.To("InternalId"),
		MtlsAuthenticationEndpoint: ptr.To("MtlsAuthenticationEndpoint"),
		NotAfter:                   ptr.To("NotAfter"),
		NotBefore:                  ptr.To("NotBefore"),
		ObjectId:                   ptr.To("ObjectId"),
		RenewAfter:                 ptr.To("RenewAfter"),
		TenantId:                   ptr.To("TenantId"),
	}
}

func delegatedResource(implicitIdentity *dataplane.UserAssignedIdentityCredentials, explicitIdentities ...dataplane.UserAssignedIdentityCredentials) dataplane.DelegatedResource {
	return dataplane.DelegatedResource{
		DelegationId:  ptr.To("DelegationId"),
		DelegationUrl: ptr.To("DelegationUrl"),
		ExplicitIdentities: func() *[]dataplane.UserAssignedIdentityCredentials {
			if len(explicitIdentities) > 0 {
				return &explicitIdentities
			}
			return nil
		}(),
		ImplicitIdentity: implicitIdentity,
		InternalId:       ptr.To("InternalId"),
		ResourceId:       ptr.To("ResourceId"),
	}
}

func userAssignedIdentityCredentials(censor bool) dataplane.UserAssignedIdentityCredentials {
	return dataplane.UserAssignedIdentityCredentials{
		AuthenticationEndpoint: ptr.To("AuthenticationEndpoint"),
		CannotRenewAfter:       ptr.To("CannotRenewAfter"),
		ClientId:               ptr.To("ClientId"),
		ClientSecret: func() *string {
			if censor {
				return nil
			}
			return ptr.To("ClientSecret")
		}(),
		ClientSecretUrl:            ptr.To("ClientSecretUrl"),
		CustomClaims:               ptr.To(customClaims()),
		MtlsAuthenticationEndpoint: ptr.To("MtlsAuthenticationEndpoint"),
		NotAfter:                   ptr.To("NotAfter"),
		NotBefore:                  ptr.To("NotBefore"),
		ObjectId:                   ptr.To("ObjectId"),
		RenewAfter:                 ptr.To("RenewAfter"),
		ResourceId:                 ptr.To("ResourceId"),
		TenantId:                   ptr.To("TenantId"),
	}
}

func customClaims() dataplane.CustomClaims {
	return dataplane.CustomClaims{
		XmsAzNwperimid: ptr.To([]string{"XmsAzNwperimid"}),
		XmsAzTm:        ptr.To("XmsAzTm"),
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
				return ptr.To(managedIdentityCredentials(censor, nil, nil))
			},
		},
		{
			name: "delegated resource without explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return ptr.To(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor))),
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor))),
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, no top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return ptr.To(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor)), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor)), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
				}, nil))
			},
		},
		{
			name: "delegated resource with explicit credentials, top-level explicit credentials",
			generateData: func(censor bool) (data *dataplane.ManagedIdentityCredentials) {
				return ptr.To(managedIdentityCredentials(censor, []dataplane.DelegatedResource{
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor)), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
					delegatedResource(ptr.To(userAssignedIdentityCredentials(censor)), userAssignedIdentityCredentials(censor), userAssignedIdentityCredentials(censor)),
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
