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
	v20200430 "github.com/Azure/ARO-RP/pkg/api/v20200430"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestGetAsyncOperationResult(t *testing.T) {
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
	mockClusterDocKey := "22222222-2222-2222-2222-222222222222"
	mockOpID := "11111111-1111-1111-1111-111111111111"

	type test struct {
		name           string
		mocks          func(*mock_database.MockOpenShiftClusters, *mock_database.MockAsyncOperations)
		wantStatusCode int
		wantAsync      bool
		wantResponse   func() *v20200430.OpenShiftCluster
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "operation and cluster exist in db - final result is available with content",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								PullSecret: "{}",
							},
							ServicePrincipalProfile: api.ServicePrincipalProfile{
								ClientSecret: "clientSecret",
							},
						},
					},
				}

				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						OpenShiftCluster:    clusterDoc.OpenShiftCluster,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *v20200430.OpenShiftCluster {
				return &v20200430.OpenShiftCluster{
					ID:   "fakeClusterID",
					Name: "resourceName",
					Type: "Microsoft.RedHatOpenShift/openshiftClusters",
				}
			},
		},
		{
			name: "operation exists in db, but no cluster - final result is available with no content",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name: "operation and cluster exist in db - final result is not yet available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{
						AsyncOperationID: mockOpID,
					}, nil)
			},
			wantAsync:      true,
			wantStatusCode: http.StatusAccepted,
		},
		{
			name: "operation not found in db",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      `404: NotFound: : The entity was not found.`,
		},
		{
			name: "internal error",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(nil, errors.New("random error"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantError:      `500: InternalServerError: : Internal server error.`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer cli.CloseIdleConnections()

			l := listener.NewListener()
			defer l.Close()

			env := &env.Test{
				L:            l,
				TestLocation: "eastus",
				TLSKey:       serverkey,
				TLSCerts:     servercerts,
			}
			env.SetARMClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))

			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			asyncOperations := mock_database.NewMockAsyncOperations(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)

			tt.mocks(openshiftClusters, asyncOperations)

			f, err := newTestFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				AsyncOperations:   asyncOperations,
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			referer := fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationresults/%s", mockSubID, env.Location(), mockOpID)

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(
				"https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/operationresults/%s?api-version=2020-04-30",
				mockSubID,
				env.Location(),
				mockOpID,
			), nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = http.Header{
				"Content-Type": []string{"application/json"},
				"Referer":      []string{referer},
			}
			resp, err := cli.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			location := resp.Header.Get("Location")
			if tt.wantAsync {
				if location != referer {
					t.Error(location)
				}
			} else {
				if location != "" {
					t.Error(location)
				}
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				if tt.wantResponse != nil {
					var oc *v20200430.OpenShiftCluster
					err = json.Unmarshal(b, &oc)
					if err != nil {
						t.Fatal(err)
					}

					if !reflect.DeepEqual(oc, tt.wantResponse()) {
						b, _ := json.Marshal(oc)
						t.Error(string(b))
					}
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
}
