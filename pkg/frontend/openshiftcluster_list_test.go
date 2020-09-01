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
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_clusterdata "github.com/Azure/ARO-RP/pkg/util/mocks/clusterdata"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_cosmosdb "github.com/Azure/ARO-RP/pkg/util/mocks/database/cosmosdb"
	mock_encryption "github.com/Azure/ARO-RP/pkg/util/mocks/encryption"
)

func TestListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		mocks          func(*gomock.Controller, *mock_database.MockOpenShiftClusters, *mock_clusterdata.MockOpenShiftClusterEnricher, *mock_encryption.MockCipher, string)
		skipToken      string
		wantStatusCode int
		wantResponse   *v20200430.OpenShiftClusterList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
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
							ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2", mockSubID),
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
				mockIter.EXPECT().Continuation().Return("")

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "").
					Return(mockIter, nil)

				enricher.EXPECT().Enrich(gomock.Any(), clusterDocs[0].OpenShiftCluster, clusterDocs[1].OpenShiftCluster)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20200430.OpenShiftClusterList{
				OpenShiftClusters: []*v20200430.OpenShiftCluster{
					{
						ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
						Name: "resourceName1",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
					{
						ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName2", mockSubID),
						Name: "resourceName2",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				},
			},
		},
		{
			name: "clusters exists in db - multiple pages",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
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
				}

				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(&api.OpenShiftClusterDocuments{OpenShiftClusterDocuments: clusterDocs}, nil)
				mockIter.EXPECT().Continuation().Return("mock-skip-token")
				cipher.EXPECT().Encrypt([]byte("mock-skip-token")).Return([]byte("encrypted-mock-skip-token"), nil)

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "").
					Return(mockIter, nil)

				enricher.EXPECT().Enrich(gomock.Any(), clusterDocs[0].OpenShiftCluster)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20200430.OpenShiftClusterList{
				OpenShiftClusters: []*v20200430.OpenShiftCluster{
					{
						ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
						Name: "resourceName1",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				},
				NextLink: "https://mockrefererhost/?%24skipToken=ZW5jcnlwdGVkLW1vY2stc2tpcC10b2tlbg%3D%3D",
			},
		},
		{
			name: "request has pagination token",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
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
				}

				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(&api.OpenShiftClusterDocuments{OpenShiftClusterDocuments: clusterDocs}, nil)
				mockIter.EXPECT().Continuation().Return("")
				cipher.EXPECT().Decrypt([]byte("encrypted-mock-skip-token")).Return([]byte("mock-skip-token"), nil)

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "mock-skip-token").
					Return(mockIter, nil)

				enricher.EXPECT().Enrich(gomock.Any(), clusterDocs[0].OpenShiftCluster)
			},
			skipToken:      "ZW5jcnlwdGVkLW1vY2stc2tpcC10b2tlbg%3D%3D",
			wantStatusCode: http.StatusOK,
			wantResponse: &v20200430.OpenShiftClusterList{
				OpenShiftClusters: []*v20200430.OpenShiftCluster{
					{
						ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
						Name: "resourceName1",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				},
			},
		},
		{
			name: "no clusters found in db",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(nil, nil)
				mockIter.EXPECT().Continuation().Return("")

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "").
					Return(mockIter, nil)

				enricher.EXPECT().Enrich(gomock.Any(), []*api.OpenShiftCluster{})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: &v20200430.OpenShiftClusterList{
				OpenShiftClusters: []*v20200430.OpenShiftCluster{},
			},
		},
		{
			name: "internal error on list",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "").
					Return(nil, errors.New("random error"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
		{
			name: "internal error while iterating list",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, enricher *mock_clusterdata.MockOpenShiftClusterEnricher, cipher *mock_encryption.MockCipher, listPrefix string) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any(), 10).Return(nil, errors.New("random error"))

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix, "").
					Return(mockIter, nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for name, listPrefix := range map[string]string{
				"subscription list":   fmt.Sprintf("/subscriptions/%s/", mockSubID),
				"resource group list": fmt.Sprintf("/subscriptions/%s/resourcegroups/resourcegroup/", mockSubID),
			} {
				t.Run(name, func(t *testing.T) {
					ti, err := newTestInfra(t)
					if err != nil {
						t.Fatal(err)
					}
					defer ti.done()

					controller := gomock.NewController(t)
					defer controller.Finish()

					dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(ti.controller)
					enricher := mock_clusterdata.NewMockOpenShiftClusterEnricher(ti.controller)
					cipher := mock_encryption.NewMockCipher(ti.controller)
					tt.mocks(controller, dbopenshiftclusters, enricher, cipher, listPrefix)

					f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), ti.env, nil, nil, dbopenshiftclusters, nil, ti.l, api.APIs, &noop.Noop{}, cipher, nil, clientauthorizer.NewOne(clientcerts[0].Raw), nil)
					if err != nil {
						t.Fatal(err)
					}
					f.(*frontend).ocEnricher = enricher

					go f.Run(ctx, nil, nil)

					resp, b, err := ti.request(http.MethodGet,
						fmt.Sprintf("https://server%sproviders/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2020-04-30&%%24skipToken=%s", listPrefix, tt.skipToken),
						http.Header{
							"Referer": []string{"https://mockrefererhost/"},
						}, nil)

					err = validateResponse(resp, b, tt.wantStatusCode, tt.wantError, tt.wantResponse)
					if err != nil {
						t.Error(err)
					}
				})
			}
		})
	}
}
