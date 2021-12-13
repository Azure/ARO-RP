package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/golang/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	mock_instancemetadata "github.com/Azure/ARO-RP/pkg/util/mocks/instancemetadata"
)

func TestARMRefreshOnce(t *testing.T) {
	for _, tt := range []struct {
		name     string
		azureEnv azureclient.AROEnvironment
		do       func(*http.Request) (*http.Response, error)
		wantErr  string
	}{
		{
			name:     "valid public cloud",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json; charset=utf-8"},
					},
					Body: ioutil.NopCloser(strings.NewReader(
						`{
							"clientCertificates": [
								{
									"notBefore": "2020-01-19T23:00:00Z",
									"notAfter": "2020-01-20T01:00:00Z"
								}
							]
						}`,
					)),
				}, nil
			},
		},
		{
			name:     "valid gov cloud",
			azureEnv: azureclient.USGovernmentCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json; charset=utf-8"},
					},
					Body: ioutil.NopCloser(strings.NewReader(
						`{
							"clientCertificates": [
								{
									"notBefore": "2020-01-19T23:00:00Z",
									"notAfter": "2020-01-20T01:00:00Z"
								}
							]
						}`,
					)),
				}, nil
			},
		},
		{
			name:     "invalid - no certificate for now",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json; charset=utf-8"},
					},
					Body: ioutil.NopCloser(strings.NewReader(
						`{
							"clientCertificates": [
								{
									"notBefore": "2020-01-20T23:00:00Z",
									"notAfter": "2020-01-21T01:00:00Z"
								}
							]
						}`,
					)),
				}, nil
			},
			wantErr: "did not receive current certificate",
		},
		{
			name:     "invalid JSON",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: ioutil.NopCloser(strings.NewReader("not JSON")),
				}, nil
			},
			wantErr: "invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name:     "invalid - error",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("fake error")
			},
			wantErr: "fake error",
		},
		{
			name:     "invalid - status code",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       ioutil.NopCloser(nil),
				}, nil
			},
			wantErr: "unexpected status code 502",
		},
		{
			name:     "invalid - content type",
			azureEnv: azureclient.PublicCloud,
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					},
					Body: ioutil.NopCloser(nil),
				}, nil
			},
			wantErr: `unexpected content type "text/plain"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			im := mock_instancemetadata.NewMockInstanceMetadata(controller)
			im.EXPECT().Environment().AnyTimes().Return(&tt.azureEnv)

			a := &arm{
				now: func() time.Time { return time.Date(2020, 1, 20, 0, 0, 0, 0, time.UTC) },
				do: func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						return nil, fmt.Errorf("unexpected method %q", req.Method)
					}
					endpoint := strings.TrimSuffix(im.Environment().ResourceManagerEndpoint, "/") + ":24582"
					if reflect.DeepEqual(im.Environment().Environment, azure.PublicCloud) {
						endpoint = "https://admin.management.azure.com"
					}
					if req.URL.String() != endpoint+"/metadata/authentication?api-version=2015-01-01" {
						return nil, fmt.Errorf("unexpected URL %q", req.URL.String())
					}
					return tt.do(req)
				},
				im: im,
			}

			if a.IsReady() {
				t.Fatal("unexpected ready state")
			}

			err := a.refreshOnce()

			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}

			if a.IsReady() != (tt.wantErr == "") {
				t.Fatal("unexpected ready state")
			}
		})
	}
}

func TestARMIsAuthorized(t *testing.T) {
	now := time.Date(2020, 1, 20, 0, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name           string
		certs          []clientCertificate
		cs             *tls.ConnectionState
		wantReady      bool
		wantAuthorized bool
	}{
		{
			name: "leaf cert matches the client certificate",
			certs: []clientCertificate{
				{
					Certificate: []byte("current"),
					NotBefore:   now.Add(-time.Hour),
					NotAfter:    now.Add(time.Hour),
				},
			},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{
						Raw: []byte("current"),
					},
				},
			},
			wantReady:      true,
			wantAuthorized: true,
		},
		{
			name: "leaf cert matches a client certificate",
			certs: []clientCertificate{
				{
					Certificate: []byte("past"),
					NotBefore:   now.Add(-6 * time.Hour),
					NotAfter:    now.Add(-5 * time.Hour),
				},
				{
					Certificate: []byte("current"),
					NotBefore:   now.Add(-time.Hour),
					NotAfter:    now.Add(time.Hour),
				},
				{
					Certificate: []byte("future"),
					NotBefore:   now.Add(5 * time.Hour),
					NotAfter:    now.Add(6 * time.Hour),
				},
			},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{
						Raw: []byte("current"),
					},
				},
			},
			wantReady:      true,
			wantAuthorized: true,
		},
		{
			name: "leaf cert does not match past client certificate",
			certs: []clientCertificate{
				{
					Certificate: []byte("past"),
					NotBefore:   now.Add(-6 * time.Hour),
					NotAfter:    now.Add(-5 * time.Hour),
				},
			},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{
						Raw: []byte("past"),
					},
				},
			},
		},
		{
			name: "leaf cert does not match future client certificate",
			certs: []clientCertificate{
				{
					Certificate: []byte("future"),
					NotBefore:   now.Add(5 * time.Hour),
					NotAfter:    now.Add(6 * time.Hour),
				},
			},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{
						Raw: []byte("future"),
					},
				},
			},
		},
		{
			name: "non-leaf cert does not match client certificate",
			certs: []clientCertificate{
				{
					Certificate: []byte("current"),
					NotBefore:   now.Add(-time.Hour),
					NotAfter:    now.Add(time.Hour),
				},
			},
			cs: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{
						Raw: []byte("does not match"),
					},
					{
						Raw: []byte("current"),
					},
				},
			},
			wantReady: true,
		},
		{
			name: "invalid connection state - not TLS",
		},
		{
			name: "invalid connection state - no PeerCertificates",
			cs:   &tls.ConnectionState{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			im := mock_instancemetadata.NewMockInstanceMetadata(controller)
			im.EXPECT().Environment().AnyTimes().Return(&azureclient.PublicCloud)

			a := &arm{
				im: im,
				now: func() time.Time {
					return now
				},
				m: metadata{
					ClientCertificates: tt.certs,
				},
				lastSuccessfulRefresh: now,
			}

			ready := a.IsReady()
			if ready != tt.wantReady {
				t.Error(ready)
			}

			isAuthorized := a.IsAuthorized(tt.cs)
			if isAuthorized != tt.wantAuthorized {
				t.Error(isAuthorized)
			}
		})
	}
}
