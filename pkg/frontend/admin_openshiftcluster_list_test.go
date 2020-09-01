package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	admin "github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_cosmosdb "github.com/Azure/ARO-RP/pkg/util/mocks/database/cosmosdb"
	mock_encryption "github.com/Azure/ARO-RP/pkg/util/mocks/encryption"
)

func TestAdminListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		mocks          func(*gomock.Controller, *mock_database.MockOpenShiftClusters, *mock_clusterdata.MockOpenShiftClusterEnricher, *mock_encryption.MockCipher)
		wantStatusCode int
		wantResponse   *[]*admin.OpenShiftCluster
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher) {
				clusterDocs := []*api.OpenShiftClusterDocument{
					{
						OpenShiftCluster: &api.OpenShiftCluster{
							ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
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
							ID:   "/subscriptions/00000000-0000-0000-0000-000000000001/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2",
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
				mockIter.EXPECT().Next(gomock.Any(), -1).Return(&api.OpenShiftClusterDocuments{OpenShiftClusterDocuments: clusterDocs}, nil)
				mockIter.EXPECT().Next(gomock.Any(), -1).Return(nil, nil)

				oc.EXPECT().
					List().
					Return(mockIter)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &[]*admin.OpenShiftCluster{
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
		},
		{
			name: "no clusters found in db",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), -1).Return(nil, nil)

				oc.EXPECT().
					List().
					Return(mockIter)
			},
			wantStatusCode: http.StatusOK,
			wantResponse:   &[]*admin.OpenShiftCluster{},
		},
		{
			name: "internal error while iterating list",
			mocks: func(controller *gomock.Controller, oc *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), -1).Return(nil, errors.New("random error"))

				oc.EXPECT().
					List().
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

			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)
			enricher := mock_clusterdata.NewMockOpenShiftClusterEnricher(ti.controller)
			cipher := mock_encryption.NewMockCipher(ti.controller)
			tt.mocks(ti.controller, dbopenshiftclusters, enricher, cipher)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, nil, dbopenshiftclusters, nil, ti.l, api.APIs, &noop.Noop{}, cipher, nil, nil, clientauthorizer.NewOne(clientcerts[0].Raw))
			if err != nil {
				t.Fatal(err)
			}
			f.(*frontend).ocEnricher = enricher

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
		})
	}
}
