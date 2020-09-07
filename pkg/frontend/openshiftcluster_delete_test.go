package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

func TestDeleteOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		resourceID     string
		mocks          func(*test, *mock_database.MockAsyncOperations, *mock_database.MockOpenShiftClusters, *mock_database.MockSubscriptions)
		wantStatusCode int
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters, subscriptions *mock_database.MockSubscriptions) {
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

				expectAsyncOperationDocumentCreate(asyncOperations, strings.ToLower(tt.resourceID), api.ProvisioningStateDeleting)

				openShiftClusters.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
					DoAndReturn(func(ctx context.Context, key string, f func(doc *api.OpenShiftClusterDocument) error) (*api.OpenShiftClusterDocument, error) {
						doc := &api.OpenShiftClusterDocument{
							Key:      key,
							Dequeues: 1,
							OpenShiftCluster: &api.OpenShiftCluster{
								ID:   tt.resourceID,
								Name: "resourceName",
								Type: "Microsoft.RedHatOpenShift/openshiftClusters",
								Properties: api.OpenShiftClusterProperties{
									ProvisioningState: api.ProvisioningStateSucceeded,
								},
							},
						}

						// doc gets modified in the callback
						err := f(doc)

						m := (*matcher.OpenShiftClusterDocument)(&api.OpenShiftClusterDocument{
							Key: key,
							OpenShiftCluster: &api.OpenShiftCluster{
								ID:   tt.resourceID,
								Name: "resourceName",
								Type: "Microsoft.RedHatOpenShift/openshiftClusters",
								Properties: api.OpenShiftClusterProperties{
									ProvisioningState:     api.ProvisioningStateDeleting,
									LastProvisioningState: api.ProvisioningStateSucceeded,
								},
							},
						})

						if !m.Matches(doc) {
							b, _ := json.MarshalIndent(doc, "", "    ")
							t.Fatal(string(b))
						}

						return doc, err
					})
			},
			wantStatusCode: http.StatusAccepted,
			wantAsync:      true,
		},
		{
			name:       "cluster not found in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, _ *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters, _ *mock_database.MockSubscriptions) {
				openShiftClusters.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:       "internal error",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, _ *mock_database.MockAsyncOperations, openShiftClusters *mock_database.MockOpenShiftClusters, _ *mock_database.MockSubscriptions) {
				openShiftClusters.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
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

			asyncOperations := mock_database.NewMockAsyncOperations(ti.controller)
			openShiftClusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			subscriptions := mock_database.NewMockSubscriptions(ti.controller)

			tt.mocks(tt, asyncOperations, openShiftClusters, subscriptions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, &database.Database{
				AsyncOperations:   asyncOperations,
				OpenShiftClusters: openShiftClusters,
				Subscriptions:     subscriptions,
			}, api.APIs, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodDelete,
				"https://server"+tt.resourceID+"?api-version=2020-04-30",
				nil, nil)

			location := resp.Header.Get("Location")
			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(location, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationresults/", mockSubID, ti.env.Location())) {
					t.Error(location)
				}
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, ti.env.Location())) {
					t.Error(azureAsyncOperation)
				}
			} else {
				if location != "" {
					t.Error(location)
				}
				if azureAsyncOperation != "" {
					t.Error(azureAsyncOperation)
				}
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, nil)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
