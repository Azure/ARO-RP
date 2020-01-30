package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/v20191231preview"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_cosmosdb "github.com/Azure/ARO-RP/pkg/util/mocks/database/cosmosdb"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestListOpenShiftCluster(t *testing.T) {
	ctx := context.Background()

	clientkey, clientcerts, err := utiltls.GenerateKeyAndCertificate("client", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(servercerts[0])

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{clientcerts[0].Raw},
						PrivateKey:  clientkey,
					},
				},
			},
		},
	}

	mockSubID := "00000000-0000-0000-0000-000000000000"

	type test struct {
		name           string
		mocks          func(*gomock.Controller, *mock_database.MockOpenShiftClusters, string)
		wantStatusCode int
		wantResponse   func() *v20191231preview.OpenShiftClusterList
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "clusters exists in db",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, listPrefix string) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any()).Return(&api.OpenShiftClusterDocuments{
					OpenShiftClusterDocuments: []*api.OpenShiftClusterDocument{
						{
							OpenShiftCluster: &api.OpenShiftCluster{
								ID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName1", mockSubID),
								Name: "resourceName1",
								Type: "Microsoft.RedHatOpenShift/openshiftClusters",
								Properties: api.Properties{
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
								Properties: api.Properties{
									ServicePrincipalProfile: api.ServicePrincipalProfile{
										ClientSecret: "clientSecret2",
									},
								},
							},
						},
					},
				}, nil)
				mockIter.EXPECT().Next(gomock.Any()).Return(nil, nil)

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix).
					Return(mockIter, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20191231preview.OpenShiftClusterList {
				return &v20191231preview.OpenShiftClusterList{
					OpenShiftClusters: []*v20191231preview.OpenShiftCluster{
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
				}
			},
		},
		{
			name: "no clusters found in db",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, listPrefix string) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any()).Return(nil, nil)

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix).
					Return(mockIter, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20191231preview.OpenShiftClusterList {
				return &v20191231preview.OpenShiftClusterList{
					OpenShiftClusters: []*v20191231preview.OpenShiftCluster{},
				}
			},
		},
		{
			name: "internal error on list",
			mocks: func(_ *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, listPrefix string) {
				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix).
					Return(nil, errors.New("random error"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
		{
			name: "internal error while iterating list",
			mocks: func(controller *gomock.Controller, openshiftClusters *mock_database.MockOpenShiftClusters, listPrefix string) {
				mockIter := mock_cosmosdb.NewMockOpenShiftClusterDocumentIterator(controller)
				mockIter.EXPECT().Next(gomock.Any()).Return(nil, errors.New("random error"))

				openshiftClusters.EXPECT().
					ListByPrefix(mockSubID, listPrefix).
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
					defer cli.CloseIdleConnections()

					l := listener.NewListener()
					defer l.Close()

					env := &env.Test{
						L:        l,
						TLSKey:   serverkey,
						TLSCerts: servercerts,
					}
					env.SetClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))

					cli.Transport.(*http.Transport).Dial = l.Dial

					controller := gomock.NewController(t)
					defer controller.Finish()

					openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
					tt.mocks(controller, openshiftClusters, listPrefix)

					f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{OpenShiftClusters: openshiftClusters}, api.APIs, &noop.Noop{})
					if err != nil {
						t.Fatal(err)
					}

					go f.Run(ctx, nil, nil)

					resp, err := cli.Get(fmt.Sprintf("https://server%sproviders/Microsoft.RedHatOpenShift/openShiftClusters?api-version=2019-12-31-preview", listPrefix))
					if err != nil {
						t.Fatal(err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != tt.wantStatusCode {
						t.Error(resp.StatusCode)
					}

					b, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						t.Fatal(err)
					}

					if tt.wantError == "" {
						var oc *v20191231preview.OpenShiftClusterList
						err = json.Unmarshal(b, &oc)
						if err != nil {
							t.Fatal(err)
						}

						if !reflect.DeepEqual(oc, tt.wantResponse()) {
							b, _ := json.Marshal(oc)
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
				})
			}
		})
	}
}
