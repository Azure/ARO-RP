package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestAdminRedeployVM(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
	mockTenantID := "00000000-0000-0000-0000-000000000000"

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

	type test struct {
		name           string
		resourceID     string
		vmName         string
		mocks          func(*test, *mock_adminactions.MockInterface, *mock_database.MockOpenShiftClusters, *mock_database.MockSubscriptions)
		wantStatusCode int
		wantResponse   func() []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			vmName:     "aro-worker-australiasoutheast-7tcq7",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, a *mock_adminactions.MockInterface, oc *mock_database.MockOpenShiftClusters, s *mock_database.MockSubscriptions) {
				clusterDoc := &api.OpenShiftClusterDocument{
					Key: tt.resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							ClusterProfile: api.ClusterProfile{
								ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", mockSubID),
							},
						},
					},
				}
				subscriptionDoc := &api.SubscriptionDocument{
					ID: mockSubID,
					Subscription: &api.Subscription{
						State: api.SubscriptionStateRegistered,
						Properties: &api.SubscriptionProperties{
							TenantID: mockTenantID,
						},
					},
				}

				oc.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)

				s.EXPECT().Get(gomock.Any(), strings.ToLower(mockSubID)).
					Return(subscriptionDoc, nil)

				a.EXPECT().VMRedeployAndWait(gomock.Any(), tt.vmName).Return(nil)

			},
			wantStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer cli.CloseIdleConnections()
			l := listener.NewListener()
			defer l.Close()
			_env := &env.Test{
				L:            l,
				TestLocation: "eastus",
				TLSKey:       serverkey,
				TLSCerts:     servercerts,
			}
			_env.SetAdminClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))
			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			a := mock_adminactions.NewMockInterface(controller)
			dbopenshiftclusters := mock_database.NewMockOpenShiftClusters(controller)
			dbsubscriptions := mock_database.NewMockSubscriptions(controller)
			tt.mocks(tt, a, dbopenshiftclusters, dbsubscriptions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), _env, nil, dbopenshiftclusters, dbsubscriptions, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster,
				*api.SubscriptionDocument) (adminactions.Interface, error) {
				return a, nil
			})

			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)
			url := fmt.Sprintf("https://server/admin%s/redeployvm?vmName=%s", tt.resourceID, tt.vmName)
			req, err := http.NewRequest(http.MethodPost, url, nil)
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

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if tt.wantError == "" {
				if tt.wantResponse != nil {
					if !bytes.Equal(b, tt.wantResponse()) {
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
