package kubeconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
	mock_proxy "github.com/Azure/ARO-RP/pkg/util/mocks/proxy"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/listener"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

// fakeServer returns a test listener for an HTTPS server which validates its
// client and echos back the request it received
func fakeServer(cacerts []*x509.Certificate, serverkey *rsa.PrivateKey, servercerts []*x509.Certificate) *listener.Listener {
	l := listener.NewListener()

	pool := x509.NewCertPool()
	pool.AddCert(cacerts[0])

	go func() {
		_ = http.Serve(tls.NewListener(l, &tls.Config{
			Certificates: []tls.Certificate{
				{
					Certificate: [][]byte{
						servercerts[0].Raw,
					},
					PrivateKey: serverkey,
				},
			},
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  pool,
		}), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Add("X-Authenticated-Name", r.TLS.PeerCertificates[0].Subject.CommonName)
			b, _ := httputil.DumpRequest(r, true)
			_, _ = w.Write(b)
		}))
	}()

	return l
}

func testKubeconfig(cacerts []*x509.Certificate, clientkey *rsa.PrivateKey, clientcerts []*x509.Certificate) ([]byte, error) {
	kc := &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{},
		},
	}

	var err error
	kc.AuthInfos[0].AuthInfo.ClientKeyData, err = utiltls.PrivateKeyAsBytes(clientkey)
	if err != nil {
		return nil, err
	}

	kc.AuthInfos[0].AuthInfo.ClientCertificateData, err = utiltls.CertAsBytes(clientcerts[0])
	if err != nil {
		return nil, err
	}

	kc.Clusters[0].Cluster.CertificateAuthorityData, err = utiltls.CertAsBytes(cacerts[0])
	if err != nil {
		return nil, err
	}

	return json.Marshal(kc)
}

func TestProxy(t *testing.T) {
	ctx := context.Background()
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster"
	username := "username"
	token := "00000000-0000-0000-0000-000000000000"
	apiServerPrivateEndpointIP := "1.2.3.4"

	cakey, cacerts, err := utiltls.GenerateKeyAndCertificate("ca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("kubernetes", cakey, cacerts[0], false, false)
	if err != nil {
		t.Fatal(err)
	}

	sreClientkey, sreClientcerts, err := utiltls.GenerateKeyAndCertificate("system:aro-sre", cakey, cacerts[0], false, true)
	if err != nil {
		t.Fatal(err)
	}

	sreKubeconfig, err := testKubeconfig(cacerts, sreClientkey, sreClientcerts)
	if err != nil {
		t.Fatal(err)
	}

	serviceClientkey, serviceClientcerts, err := utiltls.GenerateKeyAndCertificate("system:aro-service", cakey, cacerts[0], false, true)
	if err != nil {
		t.Fatal(err)
	}

	serviceKubeconfig, err := testKubeconfig(cacerts, serviceClientkey, serviceClientcerts)
	if err != nil {
		t.Fatal(err)
	}

	l := fakeServer(cacerts, serverkey, servercerts)
	defer l.Close()

	for _, tt := range []struct {
		name           string
		r              func(*http.Request)
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient, *cosmosdb.FakePortalDocumentClient)
		mocks          func(*mock_proxy.MockDialer)
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "success - elevated",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username: username,
						ID:       resourceID,
						Kubeconfig: &api.Kubeconfig{
							Elevated: true,
						},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
				openShiftClusterDocument := &api.OpenShiftClusterDocument{
					ID:  resourceID,
					Key: resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: apiServerPrivateEndpointIP,
							},
							AROServiceKubeconfig: api.SecureBytes(serviceKubeconfig),
							AROSREKubeconfig:     api.SecureBytes(sreKubeconfig),
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			mocks: func(dialer *mock_proxy.MockDialer) {
				dialer.EXPECT().DialContext(gomock.Any(), "tcp", apiServerPrivateEndpointIP+":6443").Return(l.DialContext(ctx, "", ""))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "GET /test HTTP/1.1\r\nHost: kubernetes:6443\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\nX-Authenticated-Name: system:aro-service\r\n\r\n",
		},
		{
			name: "success - not elevated",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
				openShiftClusterDocument := &api.OpenShiftClusterDocument{
					ID:  resourceID,
					Key: resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: apiServerPrivateEndpointIP,
							},
							AROServiceKubeconfig: api.SecureBytes(serviceKubeconfig),
							AROSREKubeconfig:     api.SecureBytes(sreKubeconfig),
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			mocks: func(dialer *mock_proxy.MockDialer) {
				dialer.EXPECT().DialContext(gomock.Any(), "tcp", apiServerPrivateEndpointIP+":6443").Return(l.DialContext(ctx, "", ""))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "GET /test HTTP/1.1\r\nHost: kubernetes:6443\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\nX-Authenticated-Name: system:aro-sre\r\n\r\n",
		},
		{
			name: "no auth",
			r: func(r *http.Request) {
				r.Header.Del("Authorization")
			},
			wantStatusCode: http.StatusForbidden,
			wantBody:       "Forbidden\n",
		},
		{
			name: "bad auth, not uuid",
			r: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer bad")
			},
			wantStatusCode: http.StatusForbidden,
			wantBody:       "Forbidden\n",
		},
		{
			name:           "bad auth",
			wantStatusCode: http.StatusForbidden,
			wantBody:       "Forbidden\n",
		},
		{
			name: "not kubeconfig record",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username: username,
						ID:       resourceID,
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
			},
			wantStatusCode: http.StatusForbidden,
			wantBody:       "Forbidden\n",
		},
		{
			name: "bad path",
			r: func(r *http.Request) {
				r.URL.Path = "/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/kubeconfig/proxy/test"
			},
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Bad Request\n",
		},
		{
			name: "mismatched path",
			r: func(r *http.Request) {
				r.URL.Path = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/mismatch/kubeconfig/proxy/test"
			},
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Bad Request\n",
		},
		{
			name: "sad portal database",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalClient.SetError(fmt.Errorf("sad"))
			},
			wantStatusCode: http.StatusForbidden,
			wantBody:       "Forbidden\n",
		},
		{
			name: "sad openshift database",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)

				openShiftClustersClient.SetError(fmt.Errorf("sad"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "Internal Server Error\n",
		},
		{
			name: "nil kubeconfig",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient, portalClient *cosmosdb.FakePortalDocumentClient) {
				portalDocument := &api.PortalDocument{
					ID:  token,
					TTL: 21600,
					Portal: &api.Portal{
						Username:   username,
						ID:         resourceID,
						Kubeconfig: &api.Kubeconfig{},
					},
				}
				fixture.AddPortalDocuments(portalDocument)
				checker.AddPortalDocuments(portalDocument)
				openShiftClusterDocument := &api.OpenShiftClusterDocument{
					ID:  resourceID,
					Key: resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: apiServerPrivateEndpointIP,
							},
						},
					},
				}
				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "Internal Server Error\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dbPortal, portalClient := testdatabase.NewFakePortal()
			dbOpenShiftClusters, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()

			fixture := testdatabase.NewFixture().
				WithOpenShiftClusters(dbOpenShiftClusters).
				WithPortal(dbPortal)

			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, openShiftClustersClient, portalClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			r, err := http.NewRequest(http.MethodGet,
				"https://localhost:8444"+resourceID+"/kubeconfig/proxy/test", nil)
			if err != nil {
				panic(err)
			}

			r.Header.Set("Authorization", "Bearer "+token)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			_env := mock_env.NewMockInterface(ctrl)
			_env.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)
			_env.EXPECT().Hostname().AnyTimes().Return("testhost")
			_env.EXPECT().Location().AnyTimes().Return("eastus")

			dialer := mock_proxy.NewMockDialer(ctrl)
			if tt.mocks != nil {
				tt.mocks(dialer)
			}

			_, audit := testlog.NewAudit()
			_, baseLog := testlog.New()
			_, baseAccessLog := testlog.New()
			k := New(baseLog, audit, _env, baseAccessLog, nil, nil, dbOpenShiftClusters, dbPortal, dialer)

			unauthenticatedRouter := &mux.Router{}
			unauthenticatedRouter.Use(middleware.Bearer(k.DbPortal))
			unauthenticatedRouter.Use(middleware.Log(k.Env, audit, k.BaseAccessLog))

			unauthenticatedRouter.PathPrefix("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/microsoft.redhatopenshift/openshiftclusters/{resourceName}/kubeconfig/proxy/").Handler(k.ReverseProxy)

			if tt.r != nil {
				tt.r(r)
			}

			w := responsewriter.New(r)

			unauthenticatedRouter.ServeHTTP(w, r)

			openShiftClustersClient.SetError(nil)
			portalClient.SetError(nil)

			for _, err = range checker.CheckOpenShiftClusters(openShiftClustersClient) {
				t.Error(err)
			}

			for _, err = range checker.CheckPortals(portalClient) {
				t.Error(err)
			}

			resp := w.Response()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			if resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Error(resp.Header.Get("Content-Type"))
			}

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(b) != tt.wantBody {
				t.Errorf("%q", string(b))
			}
		})
	}
}
