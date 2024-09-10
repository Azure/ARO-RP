package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/api"
	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

func TestPlatformWorkloadIdentityIDs(t *testing.T) {
	subscriptionId := "00000000-0000-0000-0000-000000000000"
	clusterRG := "aro-cluster"

	clusterName := "aro-cluster"
	clusterId := strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionId, clusterRG, clusterName))

	identityFooName := "foo"
	identityFooResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", subscriptionId, clusterRG, identityFooName)
	identityFooClientId := "0f000f00-0f00-0f00-0f00-0f000f000f00"
	identityFooObjectId := "1f001f00-1f00-1f00-1f00-1f001f001f00"

	identityBarName := "bar"
	identityBarResourceId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s", subscriptionId, clusterRG, identityBarName)
	identityBarClientId := "0ba40ba4-0ba4-0ba4-0ba4-0ba40ba40ba4"
	identityBarObjectId := "1ba41ba4-1ba4-1ba4-1ba4-1ba41ba41ba4"

	validWIClusterDoc := &api.OpenShiftClusterDocument{
		ID:  clusterId,
		Key: clusterId,
		OpenShiftCluster: &api.OpenShiftCluster{
			Properties: api.OpenShiftClusterProperties{
				PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
						identityFooName: {
							ResourceID: identityFooResourceId,
						},
						identityBarName: {
							ResourceID: identityBarResourceId,
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	for _, tt := range []struct {
		name                              string
		doc                               *api.OpenShiftClusterDocument
		userAssignedIdentitiesClientMocks func(*mock_armmsi.MockUserAssignedIdentitiesClient)
		wantErr                           string
		wantIdentities                    *map[string]api.PlatformWorkloadIdentity
	}{
		{
			name: "error - CSP cluster",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterId,
				Key: clusterId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						ServicePrincipalProfile: &api.ServicePrincipalProfile{
							ClientID:     "asdf",
							ClientSecret: "asdf",
						},
					},
				},
			},
			wantErr: "platformWorkloadIdentityIDs called for CSP cluster",
		},
		{
			name: "error - platform workload identity has invalid resource id",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterId,
				Key: clusterId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								"invalid": {
									ResourceID: "I am not a resource ID.",
								},
							},
						},
					},
				},
			},
			wantErr: "platform workload identity 'invalid' invalid: invalid resource ID: resource id 'I am not a resource ID.' must start with '/'",
		},
		{
			name: "error - unknown error when fetching details from ARM",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterId,
				Key: clusterId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								identityFooName: {
									ResourceID: identityFooResourceId,
								},
							},
						},
					},
				},
			},
			userAssignedIdentitiesClientMocks: func(mock *mock_armmsi.MockUserAssignedIdentitiesClient) {
				mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, fmt.Errorf("some error occurred"))
			},
			wantErr: "error occured when retrieving platform workload identity 'foo' details: some error occurred",
		},
		{
			name: "success - all clientIDs and objectIDs updated in clusterdoc",
			doc:  validWIClusterDoc,
			userAssignedIdentitiesClientMocks: func(mock *mock_armmsi.MockUserAssignedIdentitiesClient) {
				mock.EXPECT().Get(gomock.Any(), gomock.Eq(clusterRG), gomock.Eq(identityFooName), gomock.Any()).Times(1).
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{
						Identity: armmsi.Identity{
							Properties: &armmsi.UserAssignedIdentityProperties{
								ClientID:    &identityFooClientId,
								PrincipalID: &identityFooObjectId,
							},
						},
					}, nil)

				mock.EXPECT().Get(gomock.Any(), gomock.Eq(clusterRG), gomock.Eq(identityBarName), gomock.Any()).Times(1).
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{
						Identity: armmsi.Identity{
							Properties: &armmsi.UserAssignedIdentityProperties{
								ClientID:    &identityBarClientId,
								PrincipalID: &identityBarObjectId,
							},
						},
					}, nil)
			},
			wantIdentities: &map[string]api.PlatformWorkloadIdentity{
				identityFooName: {
					ResourceID: identityFooResourceId,
					ClientID:   identityFooClientId,
					ObjectID:   identityFooObjectId,
				},
				identityBarName: {
					ResourceID: identityBarResourceId,
					ClientID:   identityBarClientId,
					ObjectID:   identityBarObjectId,
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockUserAssignedIdentities := mock_armmsi.NewMockUserAssignedIdentitiesClient(controller)
			if tt.userAssignedIdentitiesClientMocks != nil {
				tt.userAssignedIdentitiesClientMocks(mockUserAssignedIdentities)
			}

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.doc)
			if err := fixture.Create(); err != nil {
				t.Fatal(err)
			}

			m := manager{
				log:                    logrus.NewEntry(logrus.StandardLogger()),
				doc:                    tt.doc,
				db:                     openShiftClustersDatabase,
				userAssignedIdentities: mockUserAssignedIdentities,
			}

			err := m.platformWorkloadIdentityIDs(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)

			if tt.wantIdentities != nil {
				assert.Equal(t, *tt.wantIdentities, m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities)
			}
		})
	}
}
