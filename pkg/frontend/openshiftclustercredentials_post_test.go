package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
)

func TestPostOpenShiftClusterCredentials(t *testing.T) {
	ctx := context.Background()

	apis := map[string]*api.Version{
		"2020-04-30": api.APIs["2020-04-30"],
		"no-credentials": {
			OpenShiftClusterConverter:       api.APIs["2020-04-30"].OpenShiftClusterConverter,
			OpenShiftClusterStaticValidator: api.APIs["2020-04-30"].OpenShiftClusterStaticValidator,
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		resourceID     string
		apiVersion     string
		mocks          func(*test, *mock_database.MockOpenShiftClusters)
		wantStatusCode int
		wantResponse   func(*test) *v20200430.OpenShiftClusterCredentials
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateSucceeded,
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret",
								},
								KubeadminPassword: "password",
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func(tt *test) *v20200430.OpenShiftClusterCredentials {
				return &v20200430.OpenShiftClusterCredentials{
					KubeadminUsername: "kubeadmin",
					KubeadminPassword: "password",
				}
			},
		},
		{
			name:           "credentials request is not allowed in the API version",
			resourceID:     fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			apiVersion:     "no-credentials",
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: InvalidResourceType: : The resource type 'openshiftclusters' could not be found in the namespace 'microsoft.redhatopenshift' for api version 'no-credentials'.`,
		},
		{
			name:       "cluster exists in db in creating state",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateCreating,
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret",
								},
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Creating'.`,
		},
		{
			name:       "cluster exists in db in deleting state",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateDeleting,
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret",
								},
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Deleting'.`,
		},
		{
			name:       "cluster failed to create",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateFailed,
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								FailedProvisioningState: api.ProvisioningStateCreating,
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret",
								},
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Failed'.`,
		},
		{
			name:       "cluster failed to delete",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(&api.OpenShiftClusterDocument{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   tt.resourceID,
							Name: "resourceName",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ProvisioningState: api.ProvisioningStateFailed,
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								FailedProvisioningState: api.ProvisioningStateDeleting,
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret",
								},
							},
						},
					}, nil)
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      `400: RequestNotAllowed: : Request is not allowed in provisioningState 'Failed'.`,
		},
		{
			name:       "cluster not found in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: ResourceNotFound: : The Resource 'openshiftclusters/resourcename' under resource group 'resourcegroup' was not found.`,
		},
		{
			name:       "internal error",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters) {
				openshiftClusters.EXPECT().
					Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(nil, errors.New("random error"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := newTestInfra(t)
			if err != nil {
				t.Fatal(err)
			}
			defer ti.done()

			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			dbsubscriptions := mock_database.NewMockSubscriptions(ti.controller)

			if tt.mocks != nil {
				dbsubscriptions.EXPECT().
					Get(gomock.Any(), mockSubID).
					Return(&api.SubscriptionDocument{
						Subscription: &api.Subscription{
							State: api.SubscriptionStateRegistered,
							Properties: &api.SubscriptionProperties{
								TenantID: "11111111-1111-1111-1111-111111111111",
							},
						},
					}, nil)

				tt.mocks(tt, dbopenshiftclusters)
			}

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, nil, dbopenshiftclusters, dbsubscriptions, ti.l, apis, &noop.Noop{}, nil, nil, clientauthorizer.NewOne(clientcerts[0].Raw), nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			reqAPIVersion := "2020-04-30"
			if tt.apiVersion != "" {
				reqAPIVersion = tt.apiVersion
			}

			resp, b, err := ti.request(http.MethodPost,
				fmt.Sprintf("https://server%s/listcredentials?api-version=%s", tt.resourceID, reqAPIVersion),
				nil, nil)

			var wantResponse interface{}
			if tt.wantResponse != nil {
				wantResponse = tt.wantResponse(tt)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, wantResponse)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
