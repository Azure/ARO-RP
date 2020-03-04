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
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
	"github.com/Azure/ARO-RP/test/util/matcher"
)

func TestDeleteOpenShiftCluster(t *testing.T) {
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
		resourceID     string
		mocks          func(*test, *mock_database.MockAsyncOperations, *mock_database.MockBilling, *mock_database.MockOpenShiftClusters, *mock_database.MockSubscriptions)
		wantStatusCode int
		wantAsync      bool
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "cluster exists in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, asyncOperations *mock_database.MockAsyncOperations, billing *mock_database.MockBilling, openShiftClusters *mock_database.MockOpenShiftClusters, subscriptions *mock_database.MockSubscriptions) {
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
									ProvisioningState: api.ProvisioningStateDeleting,
								},
							},
						})

						if !m.Matches(doc) {
							b, _ := json.MarshalIndent(doc, "", "    ")
							t.Fatal(string(b))
						}

						return doc, err
					})

				billing.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
					Return(&api.BillingDocument{}, nil)
			},
			wantStatusCode: http.StatusAccepted,
			wantAsync:      true,
		},
		{
			name:       "cluster not found in db",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, _ *mock_database.MockAsyncOperations, _ *mock_database.MockBilling, openShiftClusters *mock_database.MockOpenShiftClusters, _ *mock_database.MockSubscriptions) {
				openShiftClusters.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
					Return(nil, &cosmosdb.Error{StatusCode: http.StatusNotFound})
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name:       "internal error",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, _ *mock_database.MockAsyncOperations, _ *mock_database.MockBilling, openShiftClusters *mock_database.MockOpenShiftClusters, _ *mock_database.MockSubscriptions) {
				openShiftClusters.EXPECT().
					Patch(gomock.Any(), strings.ToLower(tt.resourceID), gomock.Any()).
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
				L:        l,
				TLSKey:   serverkey,
				TLSCerts: servercerts,
			}
			env.SetARMClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))

			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			asyncOperations := mock_database.NewMockAsyncOperations(controller)
			billing := mock_database.NewMockBilling(controller)
			openShiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			subscriptions := mock_database.NewMockSubscriptions(controller)

			tt.mocks(tt, asyncOperations, billing, openShiftClusters, subscriptions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				AsyncOperations:   asyncOperations,
				Billing:           billing,
				OpenShiftClusters: openShiftClusters,
				Subscriptions:     subscriptions,
			}, api.APIs, &noop.Noop{})
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			req, err := http.NewRequest(http.MethodDelete, "https://server"+tt.resourceID+"?api-version=2019-12-31-preview", nil)
			if err != nil {
				t.Fatal(err)
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
			azureAsyncOperation := resp.Header.Get("Azure-AsyncOperation")
			if tt.wantAsync {
				if !strings.HasPrefix(location, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationresults/", mockSubID, env.Location())) {
					t.Error(location)
				}
				if !strings.HasPrefix(azureAsyncOperation, fmt.Sprintf("/subscriptions/%s/providers/microsoft.redhatopenshift/locations/%s/operationsstatus/", mockSubID, env.Location())) {
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

			if tt.wantError != "" {
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

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
