package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	admin "github.com/Azure/ARO-RP/pkg/api/admin"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

type dummyOpenShiftClusterValidator struct{}

func (*dummyOpenShiftClusterValidator) Static(interface{}, *api.OpenShiftCluster) error {
	return nil
}

func expectAsyncOperationDocumentCreate(asyncOperations *mock_database.MockAsyncOperations, key string, provisioningState api.ProvisioningState) {
	asyncOperations.EXPECT().
		Create(gomock.Any(), (*matcher.AsyncOperationDocument)(
			&api.AsyncOperationDocument{
				OpenShiftClusterKey: key,
				AsyncOperation: &api.AsyncOperation{
					InitialProvisioningState: provisioningState,
					ProvisioningState:        provisioningState,
				},
			}),
		)
}

func TestPutOrPatchOpenShiftClusterAdminAPI(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"admin": {
			OpenShiftClusterConverter: api.APIs["admin"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator: func(string, string, deployment.Mode, string) api.OpenShiftClusterStaticValidator {
				return &dummyOpenShiftClusterValidator{}
			},
			OpenShiftClusterCredentialsConverter: api.APIs["admin"].OpenShiftClusterCredentialsConverter,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		resourceID     string
		request        func(*admin.OpenShiftCluster)
		isPatch        bool
		mocks          func(*test, *mock_database.MockAsyncOperations, *mock_database.MockOpenShiftClusters)
		wantEnriched   []string
		wantStatusCode int
		wantResponse   func(*test) *admin.OpenShiftCluster
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "patch with empty request",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *admin.OpenShiftCluster) {
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateAdminUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   tt.resourceID,
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
					},
				}
			},
		},
		{
			name:       "patch a cluster with registry profile should ignore registry profile",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *admin.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
				oc.Name = "resourceName"
				oc.Properties.RegistryProfiles = []admin.RegistryProfile{
					{
						Name:     "TestUser",
						Username: "TestUserName",
					},
				}
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateAdminUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateAdminUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *admin.OpenShiftCluster {
				return &admin.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: admin.OpenShiftClusterProperties{
						ProvisioningState:     admin.ProvisioningStateAdminUpdating,
						LastProvisioningState: admin.ProvisioningStateSucceeded,
						ClusterProfile: admin.ClusterProfile{
							Domain: "changed",
						},
					},
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := newTestInfra(t)
			if err != nil {
				t.Fatal(err)
			}
			defer ti.done()

			asyncOperations := mock_database.NewMockAsyncOperations(ti.controller)
			openShiftClusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			subscriptions := mock_database.NewMockSubscriptions(ti.controller)

			tt.mocks(tt, asyncOperations, openShiftClusters)

			subscriptions.EXPECT().
				Get(gomock.Any(), mockSubID).
				Return(&api.SubscriptionDocument{
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				}, nil)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, &database.Database{
				AsyncOperations:   asyncOperations,
				OpenShiftClusters: openShiftClusters,
				Subscriptions:     subscriptions,
			}, apis, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).bucketAllocator = bucket.Fixed(1)
			f.(*frontend).ocEnricher = ti.enricher

			go f.Run(ctx, nil, nil)

			oc := &admin.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			resp, b, err := ti.request(method,
				"https://server"+tt.resourceID+"?api-version=admin",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, oc)

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			var wantResponse interface{}
			if tt.wantResponse != nil {
				wantResponse = tt.wantResponse(tt)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}

}

func TestPutOrPatchOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2020-04-30": {
			OpenShiftClusterConverter: api.APIs["2020-04-30"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator: func(string, string, deployment.Mode, string) api.OpenShiftClusterStaticValidator {
				return &dummyOpenShiftClusterValidator{}
			},
			OpenShiftClusterCredentialsConverter: api.APIs["2020-04-30"].OpenShiftClusterCredentialsConverter,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		resourceID     string
		request        func(*v20200430.OpenShiftCluster)
		isPatch        bool
		mocks          func(*test, *mock_database.MockAsyncOperations, *mock_database.MockOpenShiftClusters)
		wantEnriched   []string
		wantStatusCode int
		wantResponse   func(*test) *v20200430.OpenShiftCluster
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "create a new cluster",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = "4.3.0"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateCreating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key:    strings.ToLower(tt.resourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version: "4.3.0",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Create(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusCreated,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Properties: v20200430.OpenShiftClusterProperties{
						ProvisioningState: v20200430.ProvisioningStateCreating,
						ClusterProfile: v20200430.ClusterProfile{
							Version: "4.3.0",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from succeeded",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-removed"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								PullSecret: `{"will":"be-kept"}`,
							},
							IngressProfiles: []api.IngressProfile{{Name: "will-be-removed"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "will-be-removed"}},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								ClientSecret: "will-be-kept",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								PullSecret: `{"will":"be-kept"}`,
								Domain:     "changed",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								ClientSecret: "will-be-kept",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Properties: v20200430.OpenShiftClusterProperties{
						ProvisioningState: v20200430.ProvisioningStateUpdating,
						ClusterProfile: v20200430.ClusterProfile{
							Domain: "changed",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from failed during update",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-removed"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							IngressProfiles:         []api.IngressProfile{{Name: "will-be-removed"}},
							WorkerProfiles:          []api.WorkerProfile{{Name: "will-be-removed"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateUpdating,
							LastProvisioningState:   api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Properties: v20200430.OpenShiftClusterProperties{
						ProvisioningState: v20200430.ProvisioningStateUpdating,
						ClusterProfile: v20200430.ClusterProfile{
							Domain: "changed",
						},
					},
				}
			},
		},
		{
			name:       "update a cluster from failed during creation",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateCreating,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name:       "update a cluster from failed during deletion",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateDeleting,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name:       "patch a cluster from succeeded",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
				oc.Properties.IngressProfiles = []v20200430.IngressProfile{{Name: "changed"}}
				oc.Properties.WorkerProfiles = []v20200430.WorkerProfile{{Name: "changed"}}
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateSucceeded,
							IngressProfiles:   []api.IngressProfile{{Name: "default"}},
							WorkerProfiles:    []api.WorkerProfile{{Name: "default"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:     api.ProvisioningStateUpdating,
							LastProvisioningState: api.ProvisioningStateSucceeded,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
							IngressProfiles: []api.IngressProfile{{Name: "changed"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "changed"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: v20200430.OpenShiftClusterProperties{
						ProvisioningState: v20200430.ProvisioningStateUpdating,
						ClusterProfile: v20200430.ClusterProfile{
							Domain: "changed",
						},
						IngressProfiles: []v20200430.IngressProfile{{Name: "changed"}},
						WorkerProfiles:  []v20200430.WorkerProfile{{Name: "changed"}},
					},
				}
			},
		},
		{
			name:       "patch a cluster from failed during update",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				currentClusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							IngressProfiles:         []api.IngressProfile{{Name: "will-be-kept"}},
							WorkerProfiles:          []api.WorkerProfile{{Name: "will-be-kept"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(currentClusterdoc, nil)

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateUpdating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key: strings.ToLower(tt.resourceID),
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Tags: map[string]string{"tag": "will-be-kept"},
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState:       api.ProvisioningStateUpdating,
							LastProvisioningState:   api.ProvisioningStateFailed,
							FailedProvisioningState: api.ProvisioningStateUpdating,
							ClusterProfile: api.ClusterProfile{
								Domain: "changed",
							},
							IngressProfiles: []api.IngressProfile{{Name: "will-be-kept"}},
							WorkerProfiles:  []api.WorkerProfile{{Name: "will-be-kept"}},
						},
					},
				}

				openShiftClusters.EXPECT().
					Update(gomock.Any(), (*matcher.OpenShiftClusterDocument)(clusterdoc)).
					Return(clusterdoc, nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName")},
			wantAsync:      true,
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   tt.resourceID,
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openShiftClusters",
					Tags: map[string]string{"tag": "will-be-kept"},
					Properties: v20200430.OpenShiftClusterProperties{
						ProvisioningState: v20200430.ProvisioningStateUpdating,
						ClusterProfile: v20200430.ClusterProfile{
							Domain: "changed",
						},
						IngressProfiles: []v20200430.IngressProfile{{Name: "will-be-kept"}},
						WorkerProfiles:  []v20200430.WorkerProfile{{Name: "will-be-kept"}},
					},
				}
			},
		},
		{
			name:       "patch a cluster from failed during creation",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateCreating,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose creation failed. Delete the cluster.",
		},
		{
			name:       "patch a cluster from failed during deletion",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "changed"
			},
			isPatch: true,
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						Key: strings.ToLower(tt.resourceID),
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openShiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState:       api.ProvisioningStateFailed,
								FailedProvisioningState: api.ProvisioningStateDeleting,
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: RequestNotAllowed: : Request is not allowed on cluster whose deletion failed. Delete the cluster.",
		},
		{
			name:       "creating cluster failing when provided cluster resource group already contains a cluster",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientID = mockSubID
				oc.Properties.ClusterProfile.ResourceGroupID = fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID)
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateCreating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key:    strings.ToLower(tt.resourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version:         "4.3.0",
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID),
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(clusterdoc, &cosmosdb.Error{StatusCode: http.StatusPreconditionFailed})
				openShiftClusters.EXPECT().
					GetByClientID(gomock.Any(), clusterdoc.PartitionKey, mockSubID).
					Return(&api.OpenShiftClusterDocuments{
						Count:                     0,
						OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{},
					}, nil)
				openShiftClusters.EXPECT().
					GetByClusterResourceGroupID(gomock.Any(), clusterdoc.PartitionKey, fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID)).
					Return(&api.OpenShiftClusterDocuments{
						Count: 1,
						OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
							{
								ClusterResourceGroupIDKey: fmt.Sprintf("/subscriptions/%s/resourcegroups/aro-vjb21wca", mockSubID),
							},
						},
					}, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf("400: DuplicateResourceGroup: : The provided resource group '/subscriptions/%s/resourcegroups/aro-vjb21wca' already contains a cluster.", mockSubID),
		},
		{
			name:       "creating cluster failing when provided client ID is not unique",
			resourceID: getResourcePath(mockSubID, "resourceName"),
			request: func(oc *v20200430.OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientID = mockSubID
			},
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters) {
				openShiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateCreating)

				clusterdoc := &api.OpenShiftClusterDocument{
					Key:    strings.ToLower(tt.resourceID),
					Bucket: 1,
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   tt.resourceID,
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openShiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ProvisioningState: api.ProvisioningStateCreating,
							ClusterProfile: api.ClusterProfile{
								Version: "4.3.0",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					},
				}

				openShiftClusters.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(clusterdoc, &cosmosdb.Error{StatusCode: http.StatusPreconditionFailed})
				openShiftClusters.EXPECT().
					GetByClientID(gomock.Any(), clusterdoc.PartitionKey, mockSubID).
					Return(&api.OpenShiftClusterDocuments{
						Count: 1,
						OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
							{
								ClientIDKey: mockSubID,
							},
						},
					}, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusBadRequest,
			wantError:      fmt.Sprintf("400: DuplicateClientID: : The provided client ID '%s' is already in use by a cluster.", mockSubID),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := newTestInfra(t)
			if err != nil {
				t.Fatal(err)
			}
			defer ti.done()

			asyncOperations := mock_database.NewMockAsyncOperations(ti.controller)
			openShiftClusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			subscriptions := mock_database.NewMockSubscriptions(ti.controller)

			tt.mocks(tt, asyncOperations, openShiftClusters)

			subscriptions.EXPECT().
				Get(gomock.Any(), mockSubID).
				Return(&api.SubscriptionDocument{
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: "11111111-1111-1111-1111-111111111111",
						},
					},
				}, nil)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, &database.Database{
				AsyncOperations:   asyncOperations,
				OpenShiftClusters: openShiftClusters,
				Subscriptions:     subscriptions,
			}, apis, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).bucketAllocator = bucket.Fixed(1)
			f.(*frontend).ocEnricher = ti.enricher

			go f.Run(ctx, nil, nil)

			oc := &v20200430.OpenShiftCluster{}
			if tt.request != nil {
				tt.request(oc)
			}

			method := http.MethodPut
			if tt.isPatch {
				method = http.MethodPatch
			}

			resp, b, err := ti.request(method,
				"https://server"+tt.resourceID+"?api-version=2020-04-30",
				http.Header{
					"Content-Type": []string{"application/json"},
				}, oc)
			if err != nil {
				t.Error(err)
			}

			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			var wantResponse interface{}
			if tt.wantResponse != nil {
				wantResponse = tt.wantResponse(tt)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
