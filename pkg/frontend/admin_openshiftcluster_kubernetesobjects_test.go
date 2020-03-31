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
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	mock_kubeactions "github.com/Azure/ARO-RP/pkg/util/mocks/kubeactions"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestAdminGetKubernetesObjects(t *testing.T) {
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
		objKind        string
		objNamespace   string
		objName        string
		mocks          func(*test, *mock_database.MockOpenShiftClusters, *mock_kubeactions.MockInterface)
		wantStatusCode int
		wantResponse   func() []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:         "cluster exist in db - get",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "projX",
			objName:      "config",
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

				kactions.EXPECT().
					Get(gomock.Any(), clusterDoc.OpenShiftCluster, tt.objKind, tt.objNamespace, tt.objName).
					Return([]byte(`{"Kind": "test"}`), nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() []byte {
				return []byte(`{"Kind": "test"}` + "\n")
			},
		},
		{
			name:         "cluster exist in db - list",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "projX",
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

				kactions.EXPECT().
					List(gomock.Any(), clusterDoc.OpenShiftCluster, tt.objKind, tt.objNamespace).
					Return([]byte(`{"Kind": "test"}`), nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusOK,
			wantResponse: func() []byte {
				return []byte(`{"Kind": "test"}` + "\n")
			},
		},
		{
			name:         "no kind provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "",
			objNamespace: "projX",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_kubeactions.MockInterface) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided kind '' is invalid.",
		},
		{
			name:         "secret requested",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Secret",
			objNamespace: "projX",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_kubeactions.MockInterface) {
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
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
			env.SetAdminClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))
			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			kactions := mock_kubeactions.NewMockInterface(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			tt.mocks(tt, openshiftClusters, kactions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, kactions, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)
			url := fmt.Sprintf("https://server/admin/%s/kubernetesObjects?kind=%s&namespace=%s&name=%s", tt.resourceID, tt.objKind, tt.objNamespace, tt.objName)
			req, err := http.NewRequest(http.MethodGet, url, nil)
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

func TestValidateGetAdminKubernetesObjects(t *testing.T) {
	valid := func() url.Values {
		return url.Values{
			"kind":      []string{"Valid-kind"},
			"namespace": []string{"Valid-namespace"},
			"name":      []string{"Valid-NAME-01"},
		}
	}
	longName := strings.Repeat("x", 256)

	for _, tt := range []struct {
		name    string
		modify  func(url.Values)
		wantErr string
	}{
		{
			name: "valid",
		},
		{
			name:    "invalid kind",
			modify:  func(q url.Values) { q.Set("kind", "$") },
			wantErr: "400: InvalidParameter: : The provided kind '$' is invalid.",
		},
		{
			name:    "forbidden kind",
			modify:  func(q url.Values) { q.Set("kind", "Secret") },
			wantErr: "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			name:    "empty kind",
			modify:  func(q url.Values) { delete(q, "kind") },
			wantErr: "400: InvalidParameter: : The provided kind '' is invalid.",
		},
		{
			name:    "invalid namespace",
			modify:  func(q url.Values) { q.Set("namespace", "/") },
			wantErr: "400: InvalidParameter: : The provided namespace '/' is invalid.",
		},
		{
			name:    "invalid name",
			modify:  func(q url.Values) { q.Set("name", longName) },
			wantErr: "400: InvalidParameter: : The provided name '" + longName + "' is invalid.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			q := valid()
			if tt.modify != nil {
				tt.modify(q)
			}

			err := validateGetAdminKubernetesObjects(q)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
		})
	}
}

func TestAdminPostKubernetesObjects(t *testing.T) {
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
		objInBody      *unstructured.Unstructured
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objInBody:  &unstructured.Unstructured{},
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
				tt.objInBody.SetKind("ConfigMap")
				tt.objInBody.SetName("config")
				tt.objInBody.SetNamespace("openshift-azure-logging")
				b, _ := tt.objInBody.MarshalJSON()

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).Return(clusterDoc, nil)
				kactions.EXPECT().CreateOrUpdate(gomock.Any(), clusterDoc.OpenShiftCluster, b).Return(nil)
			},
			wantStatusCode: http.StatusOK,
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
			env.SetAdminClientAuthorizer(clientauthorizer.NewOne(clientcerts[0].Raw))
			cli.Transport.(*http.Transport).Dial = l.Dial

			controller := gomock.NewController(t)
			defer controller.Finish()

			kactions := mock_kubeactions.NewMockInterface(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			tt.mocks(tt, openshiftClusters, kactions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, &database.Database{
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, kactions, nil)
			if err != nil {
				t.Fatal(err)
			}

			buf := &bytes.Buffer{}
			err = json.NewEncoder(buf).Encode(tt.objInBody)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)
			url := fmt.Sprintf("https://server/admin%s/kubernetesObjects", tt.resourceID)
			req, err := http.NewRequest(http.MethodPost, url, buf)
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

			if tt.wantError != "" || resp.StatusCode != tt.wantStatusCode {
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
