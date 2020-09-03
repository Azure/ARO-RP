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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestGetAsyncOperationsStatus(t *testing.T) {
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
	mockOpStartTime := time.Now().Add(-time.Hour).UTC()
	mockOpEndTime := time.Now().Add(-time.Minute).UTC()

	type test struct {
		name           string
		mocks          func(*mock_database.MockOpenShiftClusters, *mock_database.MockAsyncOperations)
		wantStatusCode int
		wantResponse   func() *api.AsyncOperation
		wantError      string
	}

	for _, tt := range []*test{
		{
			name: "operation and cluster exist in db - final result is available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateUpdating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *api.AsyncOperation {
				return &api.AsyncOperation{
					ID:                "fakeOpPath",
					Name:              mockOpID,
					ProvisioningState: api.ProvisioningStateFailed,
					StartTime:         mockOpStartTime,
					EndTime:           &mockOpEndTime,
					Error: &api.CloudErrorBody{
						Code:    api.CloudErrorCodeInternalServerError,
						Message: "Some error.",
					},
				}
			},
		},
		{
			name: "operation and cluster exist in db - final result is not yet available",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateUpdating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(&api.OpenShiftClusterDocument{
						AsyncOperationID: mockOpID,
					}, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *api.AsyncOperation {
				return &api.AsyncOperation{
					ID:                "fakeOpPath",
					Name:              mockOpID,
					ProvisioningState: api.ProvisioningStateUpdating,
					StartTime:         mockOpStartTime,
				}
			},
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
			name: "operation exists in db, but no cluster",
			mocks: func(openshiftClusters *mock_database.MockOpenShiftClusters, asyncOperations *mock_database.MockAsyncOperations) {
				asyncOperations.EXPECT().
					Get(gomock.Any(), mockOpID).
					Return(&api.AsyncOperationDocument{
						ID:                  mockOpID,
						OpenShiftClusterKey: mockClusterDocKey,
						AsyncOperation: &api.AsyncOperation{
							ID:                       "fakeOpPath",
							Name:                     mockOpID,
							InitialProvisioningState: api.ProvisioningStateCreating,
							ProvisioningState:        api.ProvisioningStateFailed,
							StartTime:                mockOpStartTime,
							EndTime:                  &mockOpEndTime,
							Error: &api.CloudErrorBody{
								Code:    api.CloudErrorCodeInternalServerError,
								Message: "Some error.",
							},
						},
					}, nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), mockClusterDocKey).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() *api.AsyncOperation {
				return &api.AsyncOperation{
					ID:                "fakeOpPath",
					Name:              mockOpID,
					ProvisioningState: api.ProvisioningStateFailed,
					StartTime:         mockOpStartTime,
					EndTime:           &mockOpEndTime,
					Error: &api.CloudErrorBody{
						Code:    api.CloudErrorCodeInternalServerError,
						Message: "Some error.",
					},
				}
			},
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

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, asyncOperations, openshiftClusters, nil, api.APIs, &noop.Noop{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			resp, err := cli.Get(fmt.Sprintf(
				"https://server/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/operationsstatus/%s?api-version=2020-04-30",
				mockSubID,
				env.Location(),
				mockOpID,
			))
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
				var op *api.AsyncOperation
				err = json.Unmarshal(b, &op)
				if err != nil {
					t.Fatal(err)
				}

				if !reflect.DeepEqual(op, tt.wantResponse()) {
					b, _ := json.Marshal(op)
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
}
