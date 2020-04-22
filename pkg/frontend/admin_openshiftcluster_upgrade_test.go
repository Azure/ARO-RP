package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/kubeactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_kubeactions "github.com/Azure/ARO-RP/pkg/util/mocks/kubeactions"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestAdminUpdate(t *testing.T) {
	mockSubID := "00000000-0000-0000-0000-000000000000"
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
		mocks          func(*test, *mock_database.MockOpenShiftClusters, *mock_kubeactions.MockInterface)
		wantStatusCode int
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage test",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_kubeactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
						Properties: api.OpenShiftClusterProperties{
							AROServiceKubeconfig: api.SecureBytes(""),
						},
					},
				}

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).Return(clusterDoc, nil)
				kactions.EXPECT().ClusterUpgrade(gomock.Any()).Return(nil)
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

			kactions := mock_kubeactions.NewMockInterface(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			tt.mocks(tt, openshiftClusters, kactions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), _env, &database.Database{
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (kubeactions.Interface, error) {
				return kactions, nil
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)

			url := fmt.Sprintf("https://server/admin%s/upgrade", tt.resourceID)
			req, err := http.NewRequest(http.MethodPost, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = http.Header{
				"Content-Type": []string{"application/json"},
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

			if tt.wantError != "" {
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
