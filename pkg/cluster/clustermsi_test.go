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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	mock_msidataplane "github.com/Azure/ARO-RP/pkg/util/mocks/msidataplane"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestEnsureClusterMsiCertificate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, time.September, 29, 16, 0, 0, 0, time.UTC)

	mockGuid := "00000000-0000-0000-0000-000000000000"
	clusterRGName := "aro-cluster"
	miName := "aro-cluster-msi"
	altName := "aro-cluster-msi2"
	miResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, miName)
	altResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", mockGuid, clusterRGName, altName)
	secretName := dataplane.ManagedIdentityCredentialsStoragePrefix + mockGuid

	secretNotFoundError := autorest.DetailedError{
		StatusCode: 404,
	}

	placeholderString := "placeholder"
	placeholderTime := now.Format(time.RFC3339)
	placeholderCredentialsObject := &dataplane.ManagedIdentityCredentials{
		ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
			{
				ResourceID:       &miResourceId,
				ClientID:         &placeholderString,
				ClientSecret:     &placeholderString,
				TenantID:         &placeholderString,
				NotAfter:         &placeholderTime,
				NotBefore:        &placeholderTime,
				RenewAfter:       &placeholderTime,
				CannotRenewAfter: &placeholderTime,
			},
		},
	}
	alternateCredentialsObject := &dataplane.ManagedIdentityCredentials{
		ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
			{
				ResourceID:       &altResourceId,
				ClientID:         &placeholderString,
				ClientSecret:     &placeholderString,
				TenantID:         &placeholderString,
				NotAfter:         &placeholderTime,
				NotBefore:        &placeholderTime,
				RenewAfter:       &placeholderTime,
				CannotRenewAfter: &placeholderTime,
			},
		},
	}

	tests := []struct {
		name             string
		doc              *api.OpenShiftClusterDocument
		msiDataplaneStub func(*mock_msidataplane.MockClient)
		kvClientMocks    func(*mock_azsecrets.MockClient)
		wantErr          string
	}{
		{
			name: "error - encounter error checking for an existing certificate in the key vault",
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(azsecrets.GetSecretResponse{}, fmt.Errorf("error in GetSecret")).Times(1)
			},
			wantErr: "error in GetSecret",
		},
		{
			name: "error - encounter error in MSI dataplane",
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(&dataplane.ManagedIdentityCredentials{}, errors.New("error in msi"))
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(azsecrets.GetSecretResponse{}, secretNotFoundError).Times(1)
			},
			wantErr: "error in msi",
		},
		{
			name: "success - refresh MSI certificate in keyvault",
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(placeholderCredentialsObject, nil)
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				credBytes, _ := json.Marshal(placeholderCredentialsObject)
				credString := string(credBytes)
				getSecretResponse := azsecrets.GetSecretResponse{
					Secret: azsecrets.Secret{
						Attributes: &azsecrets.SecretAttributes{},
						Value:      &credString,
						Tags: map[string]*string{
							dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Add(-1 * time.Hour).Format(time.RFC3339)),
							dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Add(1 * time.Hour).Format(time.RFC3339)),
						},
					},
				}
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(getSecretResponse, nil).Times(1)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any(), nil).Return(azsecrets.SetSecretResponse{}, nil).Times(1)
			},
		},
		{
			name: "success - don't refresh MSI certificate in keyvault",
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
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				credBytes, _ := json.Marshal(placeholderCredentialsObject)
				credString := string(credBytes)
				getSecretResponse := azsecrets.GetSecretResponse{
					Secret: azsecrets.Secret{
						Attributes: &azsecrets.SecretAttributes{},
						Value:      &credString,
						Tags: map[string]*string{
							dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Add(1 * time.Hour).Format(time.RFC3339)),
							dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Add(2 * time.Hour).Format(time.RFC3339)),
						},
					},
				}
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(getSecretResponse, nil).Times(1)
			},
		},
		{
			name: "success - successfully create and persist certificate",
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
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(placeholderCredentialsObject, nil)
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(azsecrets.GetSecretResponse{}, secretNotFoundError).Times(1)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any(), nil).Return(azsecrets.SetSecretResponse{}, nil).Times(1)
			},
		},
		{
			name: "success - successfully update cluster identity",
			doc: &api.OpenShiftClusterDocument{
				ID: mockGuid,
				OpenShiftCluster: &api.OpenShiftCluster{
					Identity: &api.ManagedServiceIdentity{
						UserAssignedIdentities: map[string]api.UserAssignedIdentity{
							altResourceId: {},
						},
					},
				},
			},
			msiDataplaneStub: func(client *mock_msidataplane.MockClient) {
				client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(alternateCredentialsObject, nil)
			},
			kvClientMocks: func(kvclient *mock_azsecrets.MockClient) {
				credBytes, _ := json.Marshal(placeholderCredentialsObject)
				credString := string(credBytes)
				getSecretResponse := azsecrets.GetSecretResponse{
					Secret: azsecrets.Secret{
						Attributes: &azsecrets.SecretAttributes{},
						Value:      &credString,
						Tags: map[string]*string{
							dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Add(1 * time.Hour).Format(time.RFC3339)),
							dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Add(2 * time.Hour).Format(time.RFC3339)),
						},
					},
				}
				kvclient.EXPECT().GetSecret(gomock.Any(), secretName, "", nil).Return(getSecretResponse, nil).Times(1)
				kvclient.EXPECT().SetSecret(gomock.Any(), secretName, gomock.Any(), nil).Return(azsecrets.SetSecretResponse{}, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			factory := mock_msidataplane.NewMockClientFactory(controller)
			if tt.msiDataplaneStub != nil {
				client := mock_msidataplane.NewMockClient(controller)
				tt.msiDataplaneStub(client)
				factory.EXPECT().NewClient(gomock.Any()).Return(client, nil).AnyTimes()
			}

			mockKvClient := mock_azsecrets.NewMockClient(controller)
			if tt.kvClientMocks != nil {
				tt.kvClientMocks(mockKvClient)
			}

			m := manager{
				log:                     logrus.NewEntry(logrus.StandardLogger()),
				doc:                     tt.doc,
				msiDataplane:            factory,
				clusterMsiKeyVaultStore: mockKvClient,
			}

			err := m.ensureClusterMsiCertificate(ctx, now)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	now := time.Date(2025, 9, 16, 16, 0, 0, 0, time.UTC)

	renewTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
	expireTime := now.Add(1 * time.Hour).Format(time.RFC3339)

	testCases := []struct {
		name        string
		item        *azsecrets.GetSecretResponse
		currentTime time.Time
		wantBool    bool
		wantErr     bool
		errContains string
	}{
		{
			name: "success - needs refresh",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(renewTime),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(expireTime),
					},
				},
			},
			currentTime: now,
			wantBool:    true,
			wantErr:     false,
		},
		{
			name: "success - not yet refreshing time",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Add(1 * time.Hour).Format(time.RFC3339)),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Add(2 * time.Hour).Format(time.RFC3339)),
					},
				},
			},
			currentTime: now,
			wantBool:    false,
			wantErr:     false,
		},
		{
			name: "success - too late to refresh",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Add(-2 * time.Hour).Format(time.RFC3339)),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Add(-1 * time.Hour).Format(time.RFC3339)),
					},
				},
			},
			currentTime: now,
			wantBool:    false,
			wantErr:     false,
		},
		{
			name: "success - exactly at renewal time",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(now.Format(time.RFC3339)),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(expireTime),
					},
				},
			},
			currentTime: now,
			wantBool:    true,
			wantErr:     false,
		},
		{
			name: "success - exactly at expiration time",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr(renewTime),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(now.Format(time.RFC3339)),
					},
				},
			},
			currentTime: now,
			wantBool:    true,
			wantErr:     false,
		},
		{
			name:        "error - tags are nil",
			item:        &azsecrets.GetSecretResponse{Secret: azsecrets.Secret{Tags: nil}},
			currentTime: now,
			wantBool:    false,
			wantErr:     true,
			errContains: "secret tags are nil",
		},
		{
			name: "error - missing renew_after tag",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(expireTime),
					},
				},
			},
			currentTime: now,
			wantBool:    false,
			wantErr:     true,
			errContains: "missing or invalid tag: renew_after",
		},
		{
			name: "error - missing cannot_renew_after Tag",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag: pointerutils.ToPtr(renewTime),
					},
				},
			},
			currentTime: now,
			wantBool:    false,
			wantErr:     true,
			errContains: "missing or invalid tag: cannot_renew_after",
		},
		{
			name: "error - invalid tag time format",
			item: &azsecrets.GetSecretResponse{
				Secret: azsecrets.Secret{
					Tags: map[string]*string{
						dataplane.RenewAfterKeyVaultTag:       pointerutils.ToPtr("not-a-valid-time"),
						dataplane.CannotRenewAfterKeyVaultTag: pointerutils.ToPtr(expireTime),
					},
				},
			},
			currentTime: now,
			wantBool:    false,
			wantErr:     true,
			errContains: "invalid time format for tag renew_after: parsing time \"not-a-valid-time\"",
		},
	}

	m := &manager{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotBool, gotErr := m.needsRefresh(tc.item, tc.currentTime)

			if tc.wantErr {
				if gotErr == nil {
					t.Errorf("expected an error, but got nil")
				}
				if !strings.Contains(gotErr.Error(), tc.errContains) {
					t.Errorf("expected error to contain %q, but got %q", tc.errContains, gotErr.Error())
				}
			} else {
				if gotErr != nil {
					t.Errorf("did not expect an error, but got: %v", gotErr)
				}
				if gotBool != tc.wantBool {
					t.Errorf("expected bool %v, but got %v", tc.wantBool, gotBool)
				}
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

	miClientId := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	miObjectId := "99999999-9999-9999-9999-999999999999"
	placeholderString := "placeholder"
	placeholderTime := time.Now().Format(time.RFC3339)

	msiDataPlaneValidStub := func(client *mock_msidataplane.MockClient) {
		client.EXPECT().GetUserAssignedIdentitiesCredentials(gomock.Any(), gomock.Any()).Return(&dataplane.ManagedIdentityCredentials{
			ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
				{
					ClientID:   &miClientId,
					ObjectID:   &miObjectId,
					ResourceID: &miResourceId,

					ClientSecret:               &placeholderString,
					TenantID:                   &placeholderString,
					AuthenticationEndpoint:     &placeholderString,
					CannotRenewAfter:           &placeholderTime,
					ClientSecretURL:            &placeholderString,
					MtlsAuthenticationEndpoint: &placeholderString,
					NotAfter:                   &placeholderTime,
					NotBefore:                  &placeholderTime,
					RenewAfter:                 &placeholderTime,
					CustomClaims: &dataplane.CustomClaims{
						XMSAzNwperimid: []string{placeholderString},
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
	placeholderTime := time.Now().Format(time.RFC3339)
	validIdentity := dataplane.UserAssignedIdentityCredentials{
		ClientID:                   &placeholderString,
		ClientSecret:               &placeholderString,
		TenantID:                   &placeholderString,
		ResourceID:                 &placeholderString,
		AuthenticationEndpoint:     &placeholderString,
		CannotRenewAfter:           &placeholderTime,
		ClientSecretURL:            &placeholderString,
		MtlsAuthenticationEndpoint: &placeholderString,
		NotAfter:                   &placeholderTime,
		NotBefore:                  &placeholderTime,
		RenewAfter:                 &placeholderTime,
		CustomClaims: &dataplane.CustomClaims{
			XMSAzNwperimid: []string{placeholderString},
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
				ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{},
			},
			wantErr: errClusterMsiNotPresentInResponse.Error(),
		},
		{
			name: "ExplicitIdentities first element is valid, returns it",
			msiCredObj: &dataplane.ManagedIdentityCredentials{
				ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
					validIdentity,
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
