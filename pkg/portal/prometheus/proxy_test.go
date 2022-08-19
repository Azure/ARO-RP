package prometheus

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/portal/util/responsewriter"
	"github.com/Azure/ARO-RP/pkg/util/portforward"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	testdatabase "github.com/Azure/ARO-RP/test/database"
	"github.com/Azure/ARO-RP/test/util/listener"
)

type conn struct {
	net.Conn
	rw *bufio.ReadWriter
}

func (c *conn) Read(b []byte) (int, error) {
	return c.rw.Read(b)
}

// fakeServer returns a test listener for an HTTPS server which validates its
// client and forwards a SPDY request to a second HTTP server which echos back
// the request it received.
func fakeServer(cacerts []*x509.Certificate, serverkey *rsa.PrivateKey, servercerts []*x509.Certificate) *listener.Listener {
	podl := listener.NewListener()

	go func() {
		_ = http.Serve(podl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := httputil.DumpRequest(r, true)
			_, _ = w.Write(b)
		}))
	}()

	kubel := listener.NewListener()

	pool := x509.NewCertPool()
	pool.AddCert(cacerts[0])

	go func() {
		_ = http.Serve(tls.NewListener(kubel, &tls.Config{
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
			w.Header().Add(httpstream.HeaderConnection, httpstream.HeaderUpgrade)
			w.Header().Add(httpstream.HeaderUpgrade, spdy.HeaderSpdy31)

			w.WriteHeader(http.StatusSwitchingProtocols)

			hijacker := w.(http.Hijacker)
			c, buf, err := hijacker.Hijack()
			if err != nil {
				panic(err)
			}

			var mu sync.Mutex
			var dataStream, errorStream httpstream.Stream
			var dataReplySent, errorReplySent <-chan struct{}

			var serverconn httpstream.Connection
			serverconn, err = spdy.NewServerConnection(&conn{Conn: c, rw: buf}, func(stream httpstream.Stream, replySent <-chan struct{}) error {
				mu.Lock()
				defer mu.Unlock()

				switch stream.Headers().Get(corev1.StreamType) {
				case corev1.StreamTypeData:
					if dataStream != nil {
						return fmt.Errorf("dataStream already set")
					}
					dataStream = stream
					dataReplySent = replySent
				case corev1.StreamTypeError:
					if errorStream != nil {
						return fmt.Errorf("errorStream already set")
					}
					errorStream = stream
					errorReplySent = replySent
				}

				if dataStream != nil && errorStream != nil {
					go func() {
						<-dataReplySent
						<-errorReplySent

						podl.Enqueue(portforward.NewStreamConn(nil, serverconn, dataStream, errorStream))
					}()
				}

				return nil
			})
			if err != nil {
				panic(err)
			}
		}))

		podl.Close()
	}()

	return kubel
}

func testKubeconfig(cacerts []*x509.Certificate, clientkey *rsa.PrivateKey, clientcerts []*x509.Certificate) ([]byte, error) {
	kc := &clientcmdv1.Config{
		Clusters: []clientcmdv1.NamedCluster{
			{
				Cluster: clientcmdv1.Cluster{
					Server: "https://kubernetes:6443",
				},
			},
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
	resourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster"
	apiServerPrivateEndpointIP := "1.2.3.4"

	cakey, cacerts, err := utiltls.GenerateKeyAndCertificate("ca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	serverkey, servercerts, err := utiltls.GenerateKeyAndCertificate("kubernetes", cakey, cacerts[0], false, false)
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
		fixtureChecker func(*testdatabase.Fixture, *testdatabase.Checker, *cosmosdb.FakeOpenShiftClusterDocumentClient)
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "success",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				openShiftClusterDocument := &api.OpenShiftClusterDocument{
					ID:  resourceID,
					Key: resourceID,
					OpenShiftCluster: &api.OpenShiftCluster{
						Properties: api.OpenShiftClusterProperties{
							NetworkProfile: api.NetworkProfile{
								APIServerPrivateEndpointIP: apiServerPrivateEndpointIP,
							},
							AROServiceKubeconfig: api.SecureBytes(serviceKubeconfig),
						},
					},
				}

				fixture.AddOpenShiftClusterDocuments(openShiftClusterDocument)
				checker.AddOpenShiftClusterDocuments(openShiftClusterDocument)
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "GET /test HTTP/1.1\r\nHost: prometheus-k8s-0:9090\r\nAccept-Encoding: gzip\r\nUser-Agent: Go-http-client/1.1\r\n\r\n",
		},
		{
			name: "bad path",
			r: func(r *http.Request) {
				r.URL.Path = "/subscriptions/BAD/resourcegroups/rg/providers/microsoft.redhatopenshift/openshiftclusters/cluster/prometheus/test"
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Bad Request\n",
		},
		{
			name: "sad database",
			fixtureChecker: func(fixture *testdatabase.Fixture, checker *testdatabase.Checker, openShiftClustersClient *cosmosdb.FakeOpenShiftClusterDocumentClient) {
				openShiftClustersClient.SetError(fmt.Errorf("sad"))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "Internal Server Error\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dbOpenShiftClusters, openShiftClustersClient := testdatabase.NewFakeOpenShiftClusters()

			fixture := testdatabase.NewFixture().
				WithOpenShiftClusters(dbOpenShiftClusters)

			checker := testdatabase.NewChecker()

			if tt.fixtureChecker != nil {
				tt.fixtureChecker(fixture, checker, openShiftClustersClient)
			}

			err := fixture.Create()
			if err != nil {
				t.Fatal(err)
			}

			r, err := http.NewRequest(http.MethodGet,
				"https://localhost:8444"+resourceID+"/prometheus/test", nil)
			if err != nil {
				panic(err)
			}

			// Override restconfig to dial the server rather than the private endpoint IP
			rc = func(oc *api.OpenShiftCluster) (*rest.Config, error) {
				if oc.Properties.NetworkProfile.APIServerPrivateEndpointIP == "" {
					return nil, errors.New("privateEndpointIP is empty")
				}

				kubeconfig := oc.Properties.AROServiceKubeconfig
				if kubeconfig == nil {
					kubeconfig = oc.Properties.AdminKubeconfig
				}
				config, err := clientcmd.Load(kubeconfig)
				if err != nil {
					return nil, err
				}

				restconfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
				if err != nil {
					return nil, err
				}

				restconfig.Dial = l.DialContext
				return restconfig, nil
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			aadAuthenticatedRouter := &mux.Router{}

			New(logrus.NewEntry(logrus.StandardLogger()), dbOpenShiftClusters, aadAuthenticatedRouter)

			if tt.r != nil {
				tt.r(r)
			}

			w := responsewriter.New(r)

			aadAuthenticatedRouter.ServeHTTP(w, r)

			resp := w.Response()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}

			if resp.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Error(resp.Header.Get("Content-Type"))
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(b) != tt.wantBody {
				t.Errorf("%q", string(b))
			}
		})
	}
}

func TestModifyResponse(t *testing.T) {
	for _, tt := range []struct {
		name     string
		body     string
		wantbody string
	}{
		{
			name:     "makes absolute a hrefs relative",
			body:     `<html><head></head><body><a href="/foo"></a></body></html>`,
			wantbody: `<html><head></head><body><a href="./foo"></a></body></html>`,
		},
		{
			name:     "makes absolute link hrefs relative",
			body:     `<html><head></head><body><link href="/foo"/></body></html>`,
			wantbody: `<html><head></head><body><link href="./foo"/></body></html>`,
		},
		{
			name:     "makes absolute script srcs relative",
			body:     `<html><head></head><body><script src="/foo"></script></body></html>`,
			wantbody: `<html><head></head><body><script src="./foo"></script></body></html>`,
		},
		{
			name:     "makes PATH_PREFIX variable relative",
			body:     `<html><head></head><body><script>var PATH_PREFIX = "";</script></body></html>`,
			wantbody: `<html><head></head><body><script>var PATH_PREFIX = ".";</script></body></html>`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Response{
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
				Body: ioutil.NopCloser(strings.NewReader(tt.body)),
			}

			p := &prometheus{}

			err := p.modifyResponse(r)
			if err != nil {
				t.Fatal(err)
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}

			if string(body) != tt.wantbody {
				t.Errorf("%q", string(body))
			}

			length, err := strconv.Atoi(r.Header.Get("Content-Length"))
			if err != nil {
				t.Fatal(err)
			}

			if length != len(body) {
				t.Error("length mismatch")
			}
		})
	}
}
