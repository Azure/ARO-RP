package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_keyvault "github.com/Azure/ARO-RP/pkg/util/mocks/keyvault"
	mock_msidataplane "github.com/Azure/ARO-RP/pkg/util/mocks/msidataplane"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnsureClusterMsiCertificate(t *testing.T) {
	ctx := context.Background()
	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)
	secretName := dataplane.ManagedIdentityCredentialsStoragePrefix + mockGuid

	secretNotFoundError := autorest.DetailedError{
		StatusCode: 404,
	}

	placeholderString := "placeholder"
	placeholderCredentialsObject := &dataplane.ManagedIdentityCredentials{
		ExplicitIdentities: []*dataplane.UserAssignedIdentityCredentials{
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
				CustomClaims: &dataplane.CustomClaims{
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
		msiDataplaneStub func(*mock_msidataplane.MockClient)
		envMocks         func(*mock_env.MockInterface)
		kvClientMocks    func(*mock_keyvault.MockManager)
		wantErr          string
	}{
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
			kvClientMocks: func(kvclient *mock_keyvault.MockManager) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName).Times(1).Return(azkeyvault.SecretBundle{}, fmt.Errorf("error in GetSecret"))
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(&dataplane.ManagedIdentityCredentials{}, errors.New("error in msi"))
			},
			kvClientMocks: func(kvclient *mock_keyvault.MockManager) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName).Times(1).Return(azkeyvault.SecretBundle{}, secretNotFoundError)
			},
			wantErr: "error in msi",
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
			kvClientMocks: func(kvclient *mock_keyvault.MockManager) {
				credentialsObjectBuffer, err := json.Marshal(placeholderCredentialsObject)
				if err != nil {
					panic(err)
				}

				credentialsObjectString := string(credentialsObjectBuffer)
				getSecretResponse := azkeyvault.SecretBundle{
					Value: &credentialsObjectString,
				}
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName).Times(1).Return(getSecretResponse, nil)
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(placeholderCredentialsObject, nil)
			},
			envMocks: func(mockEnv *mock_env.MockInterface) {
				mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(true)
			},
			kvClientMocks: func(kvclient *mock_keyvault.MockManager) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName).Times(1).Return(azkeyvault.SecretBundle{}, secretNotFoundError)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any()).Times(1).Return(nil)
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
			mockEnv.EXPECT().FeatureIsSet(env.FeatureUseMockMsiRp).Return(false).AnyTimes()

			m := manager{
				log: logrus.NewEntry(logrus.StandardLogger()),
				doc: tt.doc,
				env: mockEnv,
			}

			factory := mock_msidataplane.NewMockClientFactory(controller)
			client := mock_msidataplane.NewMockClient(controller)
			if tt.msiDataplaneStub != nil {
				tt.msiDataplaneStub(client)
			}
			factory.EXPECT().NewClient(gomock.Any()).Return(client, nil).AnyTimes()

			m.msiDataplane = factory

			mockKvClient := mock_keyvault.NewMockManager(controller)
			if tt.kvClientMocks != nil {
				tt.kvClientMocks(mockKvClient)
			}

			m.clusterMsiKeyVaultStore = mockKvClient

			err := m.ensureClusterMsiCertificate(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
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

	miClientId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	miObjectId := "99999999-9999-9999-9999-999999999999"
	placeholderString := "placeholder"

	msiDataPlaneValidStub := func(client *mock_msidataplane.MockClient) {
		client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(&dataplane.ManagedIdentityCredentials{
			ExplicitIdentities: []*dataplane.UserAssignedIdentityCredentials{
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
					CustomClaims: &dataplane.CustomClaims{
						XMSAzNwperimid: []*string{&placeholderString},
						XMSAzTm:        &placeholderString,
					},
				},
			},
		}, nil)
	}

	for _, tt := range []struct {
		name             string
		doc              *api.OpenShiftClusterDocument
		msiDataplaneStub func(*mock_msidataplane.MockClient)
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(&dataplane.ManagedIdentityCredentials{}, errors.New("error in msi"))
			},
			wantErr: "error in msi",
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
							miResourceId: {},
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
							miResourceId: {
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
							miResourceIdIncorrectCasing: {
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
			controller := gomock.NewController(t)
			defer controller.Finish()

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.doc)
			if err := fixture.Create(); err != nil {
				t.Fatal(err)
			}

			factory := mock_msidataplane.NewMockClientFactory(controller)
			client := mock_msidataplane.NewMockClient(controller)
			if tt.msiDataplaneStub != nil {
				tt.msiDataplaneStub(client)
			}
			factory.EXPECT().NewClient(gomock.Any()).Return(client, nil).AnyTimes()

			m := manager{
				log:          logrus.NewEntry(logrus.StandardLogger()),
				doc:          tt.doc,
				db:           openShiftClustersDatabase,
				msiDataplane: factory,
			}

			err := m.clusterIdentityIDs(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantDoc != nil {
				assert.Equal(t, tt.wantDoc.OpenShiftCluster, m.doc.OpenShiftCluster)
			}
		})
	}
}

func TestGetSingleExplicitIdentity(t *testing.T) {
	placeholderString := "placeholder"
	validIdentity := dataplane.UserAssignedIdentityCredentials{
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
		CustomClaims: &dataplane.CustomClaims{
			XMSAzNwperimid: []*string{&placeholderString},
			XMSAzTm:        &placeholderString,
		},
		ObjectID: &placeholderString,
	}

	for _, tt := range []struct {
		name       string
		msiCredObj *dataplane.ManagedIdentityCredentials
		want       dataplane.UserAssignedIdentityCredentials
		wantErr    string
	}{
		{
			name:       "ExplicitIdentities nil, returns error",
			msiCredObj: &dataplane.ManagedIdentityCredentials{},
			wantErr:    errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities empty, returns error",
			msiCredObj: &dataplane.ManagedIdentityCredentials{
				ExplicitIdentities: []*dataplane.UserAssignedIdentityCredentials{},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is invalid, returns error",
			msiCredObj: &dataplane.ManagedIdentityCredentials{
				ExplicitIdentities: []*dataplane.UserAssignedIdentityCredentials{
					nil,
				},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is valid, returns it",
			msiCredObj: &dataplane.ManagedIdentityCredentials{
				ExplicitIdentities: []*dataplane.UserAssignedIdentityCredentials{
					&validIdentity,
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
