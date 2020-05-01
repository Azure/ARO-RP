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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/adminactions"
	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_database "github.com/Azure/ARO-RP/pkg/util/mocks/database"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestAdminKubernetesObjectsGetAndDelete(t *testing.T) {
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
		mocks          func(*test, *mock_database.MockOpenShiftClusters, *mock_adminactions.MockInterface)
		method         string
		wantStatusCode int
		wantResponse   func() []byte
		wantError      string
	}

	for _, tt := range []*test{
		{
			method:       http.MethodGet,
			name:         "cluster exist in db - get",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				}

				kactions.EXPECT().InitializeClients(gomock.Any()).Return(nil)
				kactions.EXPECT().
					Get(gomock.Any(), tt.objKind, tt.objNamespace, tt.objName).
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
			method:       http.MethodGet,
			name:         "cluster exist in db - list",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift-project",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				}

				kactions.EXPECT().InitializeClients(gomock.Any()).Return(nil)
				kactions.EXPECT().
					List(gomock.Any(), tt.objKind, tt.objNamespace).
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
			method:       http.MethodGet,
			name:         "no groupKind provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided groupKind '' is invalid.",
		},
		{
			method:       http.MethodGet,
			name:         "secret requested",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Secret",
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			method:       http.MethodDelete,
			name:         "cluster exist in db",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "ConfigMap",
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				}

				kactions.EXPECT().InitializeClients(gomock.Any()).Return(nil)
				kactions.EXPECT().
					Delete(gomock.Any(), tt.objKind, tt.objNamespace, tt.objName).
					Return(nil)

				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).
					Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			method:       http.MethodDelete,
			name:         "no groupKind provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided groupKind '' is invalid.",
		},
		{
			method:       http.MethodDelete,
			name:         "no name provided",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "this",
			objNamespace: "openshift-project",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "400: InvalidParameter: : The provided name '' is invalid.",
		},
		{
			method:       http.MethodDelete,
			name:         "secret requested",
			resourceID:   fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objKind:      "Secret",
			objNamespace: "openshift-project",
			objName:      "config",
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s", tt.method, tt.name), func(t *testing.T) {
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

			kactions := mock_adminactions.NewMockInterface(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			tt.mocks(tt, openshiftClusters, kactions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), _env, &database.Database{
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) adminactions.Interface {
				return kactions
			}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			go f.Run(ctx, nil, nil)
			url := fmt.Sprintf("https://server/admin%s/kubernetesObjects?kind=%s&namespace=%s&name=%s", tt.resourceID, tt.objKind, tt.objNamespace, tt.objName)
			req, err := http.NewRequest(tt.method, url, nil)
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

func TestValidateAdminKubernetesObjectsNonCustomer(t *testing.T) {
	longName := strings.Repeat("x", 256)

	for _, tt := range []struct {
		test      string
		method    string
		groupKind string
		namespace string
		name      string
		wantErr   string
	}{
		{
			test:      "valid openshift namespace",
			groupKind: "Valid-kind.openshift.io",
			namespace: "openshift",
			name:      "Valid-NAME-01",
		},
		{
			test:      "invalid customer namespace",
			groupKind: "Valid-kind.openshift.io",
			namespace: "customer",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to the provided namespace 'customer' is forbidden.",
		},
		{
			test:      "invalid groupKind",
			groupKind: "$",
			namespace: "openshift-ns",
			name:      "Valid-NAME-01",
			wantErr:   "400: InvalidParameter: : The provided groupKind '$' is invalid.",
		},
		{
			test:      "forbidden groupKind",
			groupKind: "Secret",
			namespace: "openshift-ns",
			name:      "Valid-NAME-01",
			wantErr:   "403: Forbidden: : Access to secrets is forbidden.",
		},
		{
			test:      "empty groupKind",
			namespace: "openshift-ns",
			name:      "Valid-NAME-01",
			wantErr:   "400: InvalidParameter: : The provided groupKind '' is invalid.",
		},
		{
			test:      "invalid namespace",
			groupKind: "Valid-kind.openshift.io",
			namespace: "openshift-/",
			name:      "Valid-NAME-01",
			wantErr:   "400: InvalidParameter: : The provided namespace 'openshift-/' is invalid.",
		},
		{
			test:      "invalid name",
			groupKind: "Valid-kind.openshift.io",
			namespace: "openshift-ns",
			name:      longName,
			wantErr:   "400: InvalidParameter: : The provided name '" + longName + "' is invalid.",
		},
		{
			test:      "post: empty name",
			method:    http.MethodPost,
			groupKind: "Valid-kind.openshift.io",
			namespace: "openshift-ns",
			wantErr:   "400: InvalidParameter: : The provided name '' is invalid.",
		},
		{
			test:      "delete: empty name",
			method:    http.MethodDelete,
			groupKind: "Valid-kind.openshift.io",
			namespace: "openshift-ns",
			wantErr:   "400: InvalidParameter: : The provided name '' is invalid.",
		},
	} {
		t.Run(tt.test, func(t *testing.T) {
			if tt.method == "" {
				tt.method = http.MethodGet
			}

			err := validateAdminKubernetesObjectsNonCustomer(tt.method, tt.groupKind, tt.namespace, tt.name)
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
		mocks          func(*test, *mock_database.MockOpenShiftClusters, *mock_adminactions.MockInterface)
		wantStatusCode int
		objInBody      *unstructured.Unstructured
		wantError      string
	}

	for _, tt := range []*test{
		{
			name:       "basic coverage",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objInBody: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
					"metadata": map[string]interface{}{
						"namespace": "openshift-azure-logging",
						"name":      "config",
					},
				},
			},
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				}
				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).Return(clusterDoc, nil)
				kactions.EXPECT().InitializeClients(gomock.Any()).Return(nil)
				kactions.EXPECT().CreateOrUpdate(gomock.Any(), tt.objInBody).Return(nil)
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:       "secret requested",
			resourceID: fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/Microsoft.RedHatOpenShift/openShiftClusters/resourceName", mockSubID),
			objInBody: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Secret",
				},
			},
			mocks: func(tt *test, openshiftClusters *mock_database.MockOpenShiftClusters, kactions *mock_adminactions.MockInterface) {
				clusterDoc := &api.OpenShiftClusterDocument{
					OpenShiftCluster: &api.OpenShiftCluster{
						ID:   "fakeClusterID",
						Name: "resourceName",
						Type: "Microsoft.RedHatOpenShift/openshiftClusters",
					},
				}
				openshiftClusters.EXPECT().Get(gomock.Any(), strings.ToLower(tt.resourceID)).Return(clusterDoc, nil)
			},
			wantStatusCode: http.StatusForbidden,
			wantError:      "403: Forbidden: : Access to secrets is forbidden.",
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

			kactions := mock_adminactions.NewMockInterface(controller)
			openshiftClusters := mock_database.NewMockOpenShiftClusters(controller)
			tt.mocks(tt, openshiftClusters, kactions)

			f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), _env, &database.Database{
				OpenShiftClusters: openshiftClusters,
			}, api.APIs, &noop.Noop{}, nil, func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) adminactions.Interface {
				return kactions
			}, nil, nil)
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
