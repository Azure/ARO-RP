package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/Azure/msi-dataplane/pkg/dataplane/swagger"
	"github.com/Azure/msi-dataplane/pkg/store"
	mockkvclient "github.com/Azure/msi-dataplane/pkg/store/mock_kvclient"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnsureClusterMsiCertificate(t *testing.T) {
	ctx := context.Background()
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)
	secretName := fmt.Sprintf("%s-%s", mockGuid, miName)

	secretNotFoundError := &azcore.ResponseError{
		StatusCode: 404,
	}

	msiDataPlaneNotFoundError := `failed to get credentials: Request information not available
--------------------------------------------------------------------------------
RESPONSE 404: 
ERROR CODE UNAVAILABLE
--------------------------------------------------------------------------------
Response contained no body
--------------------------------------------------------------------------------
`

	placeholderString := "placeholder"
	placeholderCredentialsObject := swagger.CredentialsObject{
		ExplicitIdentities: []*swagger.NestedCredentialsObject{
			{
				ClientID:                   &placeholderString,
				ClientSecret:               &placeholderString,
				TenantID:                   &placeholderString,
				ResourceID:                 &miResourceId,
				AuthenticationEndpoint:     &placeholderString,
				CannotRenewAfter:           &placeholderString,
				ClientSecretURL:            &placeholderString,
				MtlsAuthenticationEndpoint: &placeholderString,
				NotAfter:                   &placeholderString,
				NotBefore:                  &placeholderString,
				RenewAfter:                 &placeholderString,
				CustomClaims: &swagger.CustomClaims{
					XMSAzNwperimid: []*string{&placeholderString},
					XMSAzTm:        &placeholderString,
				},
				ObjectID: &placeholderString,
			},
		},
	}

	tests := []struct {
		name             string
		doc              *api.OpenShiftClusterDocument
		msiDataplaneStub policy.ClientOptions
		kvClientMocks    func(*mockkvclient.MockKeyVaultClient)
		wantErr          string
	}{
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							"Hi hello I'm not a valid resource ID": api.ClusterUserAssignedIdentity{},
						},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'Hi hello I'm not a valid resource ID' must start with '/'",
		},
		{
			name: "error - encounter error checking for an existing certificate in the key vault",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			kvClientMocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, gomock.Any(), gomock.Any()).Times(1).Return(azsecrets.GetSecretResponse{}, fmt.Errorf("error in GetSecret"))
			},
			wantErr: "error in GetSecret",
		},
		{
			name: "error - encounter error in MSI dataplane",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			msiDataplaneStub: policy.ClientOptions{
				Transport: dataplane.NewStub([]*dataplane.CredentialsObject{
					{
						CredentialsObject: swagger.CredentialsObject{},
					},
				}),
			},
			kvClientMocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, gomock.Any(), gomock.Any()).Times(1).Return(azsecrets.GetSecretResponse{}, secretNotFoundError)
			},
			wantErr: msiDataPlaneNotFoundError,
		},
		{
			name: "success - exit early because there is already a certificate in the key vault",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			kvClientMocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				credentialsObjectBuffer, err := placeholderCredentialsObject.MarshalJSON()
				if err != nil {
					panic(err)
				}

				credentialsObjectString := string(credentialsObjectBuffer)
				getSecretResponse := azsecrets.GetSecretResponse{
					Secret: azsecrets.Secret{
						Value: &credentialsObjectString,
					},
				}
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, gomock.Any(), gomock.Any()).Times(1).Return(getSecretResponse, nil)
			},
		},
		{
			name: "success - successfully create and persist certificate",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{
								ClientID:    mockGuid,
								PrincipalID: mockGuid,
							},
						},
					},
				},
			},
			msiDataplaneStub: policy.ClientOptions{
				Transport: dataplane.NewStub([]*dataplane.CredentialsObject{
					{
						CredentialsObject: placeholderCredentialsObject,
					},
				}),
			},
			kvClientMocks: func(kvclient *mockkvclient.MockKeyVaultClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, gomock.Any(), gomock.Any()).Times(1).Return(azsecrets.GetSecretResponse{}, secretNotFoundError)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any(), gomock.Any()).Times(1).Return(azsecrets.SetSecretResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
			}

			msiDataplane, err := dataplane.NewClient(dataplane.AzurePublicCloud, nil, &tt.msiDataplaneStub)
			if err != nil {
				panic(err)
			}

			m.msiDataplane = msiDataplane

			mockKvClient := mockkvclient.NewMockKeyVaultClient(controller)
			if tt.kvClientMocks != nil {
				tt.kvClientMocks(mockKvClient)
			}

			m.clusterMsiKeyVaultStore = store.NewMsiKeyVaultStore(mockKvClient)

			err = m.ensureClusterMsiCertificate(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestClusterMsiSecretName(t *testing.T) {
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name       string
		doc        *api.OpenShiftClusterDocument
		wantResult string
		wantErr    string
	}{
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							"Hi hello I'm not a valid resource ID": api.ClusterUserAssignedIdentity{},
						},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'Hi hello I'm not a valid resource ID' must start with '/'",
		},
		{
			name: "success",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{},
						},
					},
				},
			},
			wantResult: fmt.Sprintf("%s-%s", mockGuid, miName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
			}

			result, err := m.clusterMsiSecretName()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if result != tt.wantResult {
				t.Error(result)
			}
		})
	}
}

func TestClusterMsiResourceId(t *testing.T) {
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	tests := []struct {
		name    string
		doc     *api.OpenShiftClusterDocument
		wantErr string
	}{
		{
			name: "error - cluster doc has nil Identity",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil Identity but nil Identity.UserAssignedIdentities",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{},
				},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - cluster doc has non-nil Identity but empty Identity.UserAssignedIdentities",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{},
					},
				},
			},
			wantErr: "could not find cluster MSI in cluster doc",
		},
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							"Hi hello I'm not a valid resource ID": api.ClusterUserAssignedIdentity{},
						},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'Hi hello I'm not a valid resource ID' must start with '/'",
		},
		{
			name: "success",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.Identity{
						UserAssignedIdentities: api.UserAssignedIdentities{
							miResourceId: api.ClusterUserAssignedIdentity{},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
			}

			_, err := m.clusterMsiResourceId()
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
