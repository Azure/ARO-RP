package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/util/wait"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"

	"github.com/Azure/ARO-RP/pkg/api"
	utilarm "github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	mock_armmsi "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/armmsi"
	mock_azcore "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azcore"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
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
	fooTarget := `.properties.platformWorkloadIdentityProfile.platformWorkloadIdentities["foo"]`
	unauthorizedErr := azruntime.NewResponseError(&http.Response{StatusCode: http.StatusUnauthorized})
	forbiddenErr := azruntime.NewResponseError(&http.Response{StatusCode: http.StatusForbidden})
	authorizationErr := &sdkazcore.ResponseError{StatusCode: http.StatusBadRequest, ErrorCode: "AuthorizationFailed"}
	notFoundErr := azruntime.NewResponseError(&http.Response{StatusCode: http.StatusNotFound})
	tooManyRequestsErr := azruntime.NewResponseError(&http.Response{StatusCode: http.StatusTooManyRequests})

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

	// Must not call t.Parallel(); mutates package-level utilarm.TransientBackoff.
	savedBackoff := utilarm.TransientBackoff
	utilarm.TransientBackoff = wait.Backoff{
		Steps:    utilarm.RBACPropagationRetrySteps,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.0,
	}
	defer func() { utilarm.TransientBackoff = savedBackoff }()

	for _, tt := range []struct {
		name                              string
		doc                               *api.OpenShiftClusterDocument
		userAssignedIdentitiesClientMocks func(*mock_armmsi.MockUserAssignedIdentitiesClient)
		wantErr                           string
		wantIdentities                    *map[string]api.PlatformWorkloadIdentity
		wantRebuilds                      int
		rebuildErr                        error
		transientLookupErr                error
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
			wantErr: "error occurred when retrieving platform workload identity 'foo' details: some error occurred",
		},
		{
			name: "error - unauthorized identity lookup becomes invalid platform workload identity",
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
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, unauthorizedErr)
			},
			wantErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, unauthorizedErr.Error()).Error(),
		},
		{
			name:         "error - forbidden identity lookup becomes invalid platform workload identity",
			wantRebuilds: utilarm.RBACPropagationRetrySteps,
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
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, forbiddenErr)
			},
			wantErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, forbiddenErr.Error()).Error(),
		},
		{
			name: "error - not found identity lookup becomes invalid platform workload identity",
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
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, notFoundErr)
			},
			wantErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, notFoundErr.Error()).Error(),
		},
		{
			name: "error - too many requests identity lookup becomes invalid platform workload identity",
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
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, tooManyRequestsErr)
			},
			wantErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, tooManyRequestsErr.Error()).Error(),
		},
		{
			name: "error - failed rebuilds preserve identity error after retry exhaustion",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterId,
				Key: clusterId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								identityFooName: {ResourceID: identityFooResourceId},
							},
						},
					},
				},
			},
			userAssignedIdentitiesClientMocks: func(mock *mock_armmsi.MockUserAssignedIdentitiesClient) {
				mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(utilarm.RBACPropagationRetrySteps).
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, forbiddenErr)
			},
			wantRebuilds: utilarm.RBACPropagationRetrySteps,
			rebuildErr:   errors.New("cannot rebuild cluster MSI credential"),
			wantErr:      api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, forbiddenErr.Error()).Error(),
		},
		{
			name:               "success - failed credential rebuild does not stop identity retries",
			doc:                validWIClusterDoc,
			wantRebuilds:       1,
			rebuildErr:         errors.New("cannot rebuild cluster MSI credential"),
			transientLookupErr: forbiddenErr,
		},
		{
			name:               "success - transient forbidden error retried then succeeds",
			doc:                validWIClusterDoc,
			wantRebuilds:       1,
			transientLookupErr: forbiddenErr,
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
		{
			name:               "success - authorization error code retried without HTTP 403",
			doc:                validWIClusterDoc,
			wantRebuilds:       1,
			transientLookupErr: authorizationErr,
		},
		{
			name:               "success - unauthorized client message retried with fresh credential",
			doc:                validWIClusterDoc,
			wantRebuilds:       1,
			transientLookupErr: errors.New("AADSTS700016: application not found"),
		},
		{
			name: "error - authorization code exhausts retries as invalid platform workload identity",
			doc: &api.OpenShiftClusterDocument{
				ID:  clusterId,
				Key: clusterId,
				OpenShiftCluster: &api.OpenShiftCluster{
					Properties: api.OpenShiftClusterProperties{
						PlatformWorkloadIdentityProfile: &api.PlatformWorkloadIdentityProfile{
							PlatformWorkloadIdentities: map[string]api.PlatformWorkloadIdentity{
								identityFooName: {ResourceID: identityFooResourceId},
							},
						},
					},
				},
			},
			userAssignedIdentitiesClientMocks: func(mock *mock_armmsi.MockUserAssignedIdentitiesClient) {
				mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(utilarm.RBACPropagationRetrySteps).
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, authorizationErr)
			},
			wantRebuilds: utilarm.RBACPropagationRetrySteps,
			wantErr:      api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, authorizationErr.Error()).Error(),
		},
		{
			name: "success - transient too many requests error retried then succeeds",
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
				gomock.InOrder(
					mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, tooManyRequestsErr),
					mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(armmsi.UserAssignedIdentitiesClientGetResponse{
							Identity: armmsi.Identity{
								Properties: &armmsi.UserAssignedIdentityProperties{
									ClientID:    &identityFooClientId,
									PrincipalID: &identityFooObjectId,
								},
							},
						}, nil),
				)
			},
			wantIdentities: &map[string]api.PlatformWorkloadIdentity{
				identityFooName: {
					ResourceID: identityFooResourceId,
					ClientID:   identityFooClientId,
					ObjectID:   identityFooObjectId,
				},
			},
		},
		{
			name:         "error - forbidden exhausts all retries",
			wantRebuilds: utilarm.RBACPropagationRetrySteps,
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
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, forbiddenErr)
			},
			wantErr: api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidPlatformWorkloadIdentity, fooTarget, forbiddenErr.Error()).Error(),
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

			builds := 0
			credential := mock_azcore.NewMockTokenCredential(controller)
			refreshableCredential, err := refreshable.NewTokenCredential(func() (azcore.TokenCredential, error) {
				builds++
				if builds > 1 && tt.rebuildErr != nil {
					return nil, tt.rebuildErr
				}
				return credential, nil
			})
			require.NoError(t, err)

			mockUserAssignedIdentities := mock_armmsi.NewMockUserAssignedIdentitiesClient(controller)
			if tt.userAssignedIdentitiesClientMocks != nil {
				tt.userAssignedIdentitiesClientMocks(mockUserAssignedIdentities)
			}
			if tt.transientLookupErr != nil {
				gomock.InOrder(
					mockUserAssignedIdentities.EXPECT().Get(gomock.Any(), clusterRG, identityFooName, gomock.Any()).
						Return(armmsi.UserAssignedIdentitiesClientGetResponse{}, tt.transientLookupErr),
					mockUserAssignedIdentities.EXPECT().Get(gomock.Any(), clusterRG, identityFooName, gomock.Any()).
						Return(armmsi.UserAssignedIdentitiesClientGetResponse{
							Identity: armmsi.Identity{
								Properties: &armmsi.UserAssignedIdentityProperties{
									ClientID:    &identityFooClientId,
									PrincipalID: &identityFooObjectId,
								},
							},
						}, nil),
				)
				mockUserAssignedIdentities.EXPECT().Get(gomock.Any(), clusterRG, identityBarName, gomock.Any()).
					Return(armmsi.UserAssignedIdentitiesClientGetResponse{
						Identity: armmsi.Identity{
							Properties: &armmsi.UserAssignedIdentityProperties{
								ClientID:    &identityBarClientId,
								PrincipalID: &identityBarObjectId,
							},
						},
					}, nil)
			}

			openShiftClustersDatabase, _ := testdatabase.NewFakeOpenShiftClusters()
			fixture := testdatabase.NewFixture().WithOpenShiftClusters(openShiftClustersDatabase)
			fixture.AddOpenShiftClusterDocuments(tt.doc)
			if err := fixture.Create(); err != nil {
				t.Fatal(err)
			}

			_, log := testlog.LogForTesting(t)
			m := manager{
				log:                    log,
				doc:                    tt.doc,
				db:                     openShiftClustersDatabase,
				userAssignedIdentities: mockUserAssignedIdentities,
				clusterMsiCredential:   refreshableCredential,
			}

			err = m.platformWorkloadIdentityIDs(ctx)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			assert.Equal(t, tt.wantRebuilds, builds-1)

			if err == nil {
				err = m.persistPlatformWorkloadIdentityIDs(ctx)
				utilerror.AssertErrorMessage(t, err, tt.wantErr)
			}

			if tt.wantIdentities != nil {
				assert.Equal(t, *tt.wantIdentities, m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities)
			}
		})
	}
}
