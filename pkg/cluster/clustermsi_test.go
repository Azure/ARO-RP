package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/Azure/msi-dataplane/pkg/dataplane/swagger"
	"github.com/Azure/msi-dataplane/pkg/store"
	mockkvclient "github.com/Azure/msi-dataplane/pkg/store/mock_kvclient"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	testdatabase "github.com/Azure/ARO-RP/test/database"
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
		envMocks         func(*mock_env.MockInterface)
		kvClientMocks    func(*mockkvclient.MockKeyVaultClient)
		wantErr          string
	}{
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			doc: &api.OpenShiftClusterDocument{
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							"Hi hello I'm not a valid resource ID": {},
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
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
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
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
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
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
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
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {
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
			envMocks: func(mockEnv *mock_env.MockInterface) {
				mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(true)
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

			mockEnv := mock_env.NewMockInterface(controller)
			if tt.envMocks != nil {
				tt.envMocks(mockEnv)
			}

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
				env: mockEnv,
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
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							"Hi hello I'm not a valid resource ID": {},
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
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
						},
					},
				},
			},
			wantResult: fmt.Sprintf("%s-%s", mockGuid, miName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestClusterIdentityIDs(t *testing.T) {
	ctx := context.Background()

	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	clusterName := "aro-cluster"

	clusterResourceId := strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", mockGuid, clusterRGName, clusterName))

	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)
	miResourceIdIncorrectCasing := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)

	msiDataPlaneNotFoundError := `failed to get credentials: Request information not available
--------------------------------------------------------------------------------
RESPONSE 404: 
ERROR CODE UNAVAILABLE
--------------------------------------------------------------------------------
Response contained no body
--------------------------------------------------------------------------------
`
	miClientId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	miObjectId := "99999999-9999-9999-9999-999999999999"
	placeholderString := "placeholder"

	msiDataPlaneValidStub := policy.ClientOptions{
		Transport: dataplane.NewStub([]*dataplane.CredentialsObject{
			{
				CredentialsObject: swagger.CredentialsObject{
					ExplicitIdentities: []*swagger.NestedCredentialsObject{
						{
							ClientID:   &miClientId,
							ObjectID:   &miObjectId,
							ResourceID: &miResourceId,

							ClientSecret:               &placeholderString,
							TenantID:                   &placeholderString,
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
						},
					},
				},
			},
		}),
	}

	for _, tt := range []struct {
		name             string
		doc              *api.OpenShiftClusterDocument
		msiDataplaneStub policy.ClientOptions
		wantDoc          *api.OpenShiftClusterDocument
		wantErr          string
	}{
		{
			name: "error - CSP cluster",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							ClientID:     "asdf",
							ClientSecret: "asdf",
						},
					},
				},
			},
			wantErr: "clusterIdentityIDs called for CSP cluster",
		},
		{
			name: "error - invalid resource ID (theoretically not possible, but still)",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							"Hi hello I'm not a valid resource ID": {},
						},
					},
				},
			},
			wantErr: "invalid resource ID: resource id 'Hi hello I'm not a valid resource ID' must start with '/'",
		},
		{
			name: "error - encounter error in MSI dataplane",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: {},
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
			wantErr: msiDataPlaneNotFoundError,
		},
		{
			name: "success - ClientID and PrincipalID are updated",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: api.UserAssignedIdentity{},
						},
					},
				},
			},
			msiDataplaneStub: msiDataPlaneValidStub,
			wantDoc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceId: api.UserAssignedIdentity{
								ClientID:    miClientId,
								PrincipalID: miObjectId,
							},
						},
					},
				},
			},
		},
		{
			name: "success - existing identity resourceID casing is preserved even if it differs from ARM resourceID parsing",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceIdIncorrectCasing: {},
						},
					},
				},
			},
			msiDataplaneStub: msiDataPlaneValidStub,
			wantDoc: &api.OpenShiftClusterDocument{
				ID:  clusterResourceId,
				Key: clusterResourceId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{},
					},
					Identity: &api.ManagedServiceIdentity{
						IdentityURL: middleware.MockIdentityURL,
						TenantID:    mockGuid,
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							miResourceIdIncorrectCasing: api.UserAssignedIdentity{
								ClientID:    miClientId,
								PrincipalID: miObjectId,
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			msiDataplane, err := dataplane.NewClient(dataplane.AzurePublicCloud, nil, &tt.msiDataplaneStub)
			if err != nil {
				t.Fatal(err)
			}

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.doc)
			if err = fixture.Create(); err != nil {
				t.Fatal(err)
			}

			m := manager{
				log:          logrus.NewEntry(logrus.StandardLogger()),
				doc:          tt.doc,
				db:           openShiftClustersDatabase,
				msiDataplane: msiDataplane,
			}

			err = m.clusterIdentityIDs(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantDoc != nil {
				assert.Equal(t, tt.wantDoc.OpenShiftCluster, m.doc.OpenShiftCluster)
			}
		})
	}
}

func TestGetSingleExplicitIdentity(t *testing.T) {
	placeholderString := "placeholder"
	validIdentity := &swagger.NestedCredentialsObject{
		ClientID:                   &placeholderString,
		ClientSecret:               &placeholderString,
		TenantID:                   &placeholderString,
		ResourceID:                 &placeholderString,
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
	}

	for _, tt := range []struct {
		name       string
		msiCredObj *dataplane.UserAssignedIdentities
		want       *swagger.NestedCredentialsObject
		wantErr    string
	}{
		{
			name:       "ExplicitIdentities nil, returns error",
			msiCredObj: &dataplane.UserAssignedIdentities{},
			wantErr:    errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities empty, returns error",
			msiCredObj: &dataplane.UserAssignedIdentities{
				CredentialsObject: dataplane.CredentialsObject{
					CredentialsObject: swagger.CredentialsObject{
						ExplicitIdentities: []*swagger.NestedCredentialsObject{},
					},
				},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is nil, returns error",
			msiCredObj: &dataplane.UserAssignedIdentities{
				CredentialsObject: dataplane.CredentialsObject{
					CredentialsObject: swagger.CredentialsObject{
						ExplicitIdentities: []*swagger.NestedCredentialsObject{
							nil,
						},
					},
				},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is nil, returns error",
			msiCredObj: &dataplane.UserAssignedIdentities{
				CredentialsObject: dataplane.CredentialsObject{
					CredentialsObject: swagger.CredentialsObject{
						ExplicitIdentities: []*swagger.NestedCredentialsObject{
							nil,
						},
					},
				},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is valid, returns it",
			msiCredObj: &dataplane.UserAssignedIdentities{
				CredentialsObject: dataplane.CredentialsObject{
					CredentialsObject: swagger.CredentialsObject{
						ExplicitIdentities: []*swagger.NestedCredentialsObject{
							validIdentity,
						},
					},
				},
			},
			want: validIdentity,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSingleExplicitIdentity(tt.msiCredObj)

			assert.Equal(t, tt.want, got)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
