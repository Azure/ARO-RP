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
	"strings"
	"testing"
	"time"
)

func TestARMRefreshOnce(t *testing.T) {
	tests := []struct {
		name    string
		do      func(*http.Request) (*http.Response, error)
		wantErr string
	}{
		{
			name: "valid",
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
			name: "invalid - no certificate for now",
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
			name: "invalid JSON",
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
			name: "invalid - error",
			do: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("fake error")
			},
			wantErr: "fake error",
		},
		{
			name: "invalid - status code",
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       ioutil.NopCloser(nil),
				}, nil
			},
			wantErr: "unexpected status code 502",
		},
		{
			name: "invalid - content type",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &arm{
				now: func() time.Time { return time.Date(2020, 1, 20, 0, 0, 0, 0, time.UTC) },
				do: func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						return nil, fmt.Errorf("unexpected method %s", req.Method)
					}
					if req.URL.String() != "https://management.azure.com:24582/metadata/authentication?api-version=2015-01-01" {
						return nil, fmt.Errorf("unexpected URL %q", req.URL.String())
					}
					return tt.do(req)
				},
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

	tests := []struct {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &arm{
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
