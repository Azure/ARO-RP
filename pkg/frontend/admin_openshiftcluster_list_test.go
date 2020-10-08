package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	admin "github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_cosmosdb "github.com/Azure/ARO-RP/pkg/util/mocks/database/cosmosdb"
	mock_encryption "github.com/Azure/ARO-RP/pkg/util/mocks/encryption"
)

func TestAdminListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"
	otherMockSubID := "00000000-0000-0000-0000-000000000001"

	type test struct {
		name           string
		mocks          func(*gomock.Controller, *mock_database.MockOpenShiftClusters, *mock_encryption.MockCipher)
		wantEnriched   []string
		wantStatusCode int
		wantResponse   *admin.OpenShiftClusterList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, cipher *mock_encryption.MockCipher) {
				clusterDocs := []*api.OpenShiftClusterDocument{
					{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   getResourcePath(mockSubID, "resourceName1"),
							Name: "resourceName1",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret1",
								},
							},
						},
					},
					{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   getResourcePath(otherMockSubID, "resourceName2"),
							Name: "resourceName2",
							Type: "Microsoft.RedHatOpenShift/openshiftClusters",
							Properties: api.OpenShiftClusterProperties{
								ClusterProfile: api.ClusterProfile{
									PullSecret: "{}",
								},
								ServicePrincipalProfile: api.ServicePrincipalProfile{
									ClientSecret: "clientSecret2",
								},
							},
						},
					},
				}

				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(&api.OpenShiftClusterDocuments{OpenShiftClusterDocuments: clusterDocs}, nil)
				mockIter.EXPECT().Continuation().Return("mock-skip-token")

				oc.EXPECT().
					List("").
					Return(mockIter)

				cipher.EXPECT().
					Encrypt([]byte("mock-skip-token")).
					Return([]byte("encrypted-mock-skip-token"), nil)
			},
			wantEnriched:   []string{getResourcePath(mockSubID, "resourceName1"), getResourcePath(otherMockSubID, "resourceName2")},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftClusterList{
				OpenShiftClusters: []*admin.OpenShiftCluster{
					{
						ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
						Name: "resourceName1",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
					{
						ID:   "/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2",
						Name: "resourceName2",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				},
				NextLink: "https://mockrefererhost/?%24skipToken=ZW5jcnlwdGVkLW1vY2stc2tpcC10b2tlbg%3D%3D",
			},
		},
		{
			name: "no clusters found in db",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, cipher *mock_encryption.MockCipher) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(nil, nil)
				mockIter.EXPECT().Continuation().Return("")

				oc.EXPECT().
					List("").
					Return(mockIter)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &admin.OpenShiftClusterList{
				OpenShiftClusters: []*admin.OpenShiftCluster{},
			},
		},
		{
			name: "internal error while iterating list",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, cipher *mock_encryption.MockCipher) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(nil, errors.New("random error"))

				oc.EXPECT().
					List("").
					Return(mockIter)
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

			oc := mock_database.NewMockOpenShiftClusters(ti.controller)
			cipher := mock_encryption.NewMockCipher(ti.controller)
			tt.mocks(ti.controller, oc, cipher)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, oc, nil, api.APIs, &noop.Noop{}, cipher, nil)
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).ocEnricher = ti.enricher

			go f.Run(ctx, nil, nil)

			resp, b, err := ti.request(http.MethodGet,
				"https://server/admin/providers/Microsoft.RedHatOpenShift/openShiftClusters",
				http.Header{
					"Referer": []string{"https://mockrefererhost/"},
				}, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
			if err != nil {
				t.Error(err)
			}

			if tt.wantError == "" {
				var ocs *admin.OpenShiftClusterList
				err = json.Unmarshal(b, &ocs)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(ocs, tt.wantResponse) {
					b, _ := json.Marshal(ocs)
					t.Error(string(b))
				}

			} else {
				cloudErr := &api.CloudError{StatusCode: resp.StatusCode}
				err = json.Unmarshal(b, &cloudErr)
				if err != nil {
					t.Fatal(err)
				}

				if cloudErr.Error() != tt.wantError {
					t.Error(cloudErr)
				}
			}

			errs := ti.enricher.Check(tt.wantEnriched)
			for _, err := range errs {
				t.Error(err)
			}
		})
	}
}
