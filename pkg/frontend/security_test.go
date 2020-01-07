package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestSecurity(t *testing.T) {
	ctx := context.Background()

	validclientkey, validclientcerts, err := utiltls.GenerateKeyAndCertificate("validclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	l := listener.NewListener()
	defer l.Close()

	env := env.NewTest(l, validclientcerts[0].Raw)

	env.TLSKey, env.TLSCerts, err = utiltls.GenerateKeyAndCertificate("server", nil, nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	invalidclientkey, invalidclientcerts, err := utiltls.GenerateKeyAndCertificate("invalidclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(env.TLSCerts[0])

	f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, nil, api.APIs, &noop.Noop{})
	if err != nil {
		t.Fatal(err)
	}

	go f.Run(ctx, nil, nil)

	for _, tt := range []struct {
		name           string
		url            string
		key            *rsa.PrivateKey
		cert           *x509.Certificate
		wantStatusCode int
	}{
		{
			name:           "empty url, no client certificate",
			url:            "https://server/",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, no client certificate",
			url:            "https://server/unknown",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, no client certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2019-12-31-preview",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ready url, no client certificate",
			url:            "https://server/healthz/ready",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty url, invalid certificate",
			url:            "https://server/",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, invalid certificate",
			url:            "https://server/unknown",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, invalid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2019-12-31-preview",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ready url, invalid certificate",
			url:            "https://server/healthz/ready",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty url, valid certificate",
			url:            "https://server/",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "unknown url, valid certificate",
			url:            "https://server/unknown",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "operations url, valid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2019-12-31-preview",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "ready url, valid certificate",
			url:            "https://server/healthz/ready",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig := &tls.Config{
				RootCAs: pool,
			}
			if tt.cert != nil && tt.key != nil {
				tlsConfig.Certificates = []tls.Certificate{
					{
						Certificate: [][]byte{
							tt.cert.Raw,
						},
						PrivateKey: tt.key,
					},
				}
			}

			c := &http.Client{
				Transport: &http.Transport{
					Dial:            l.Dial,
					TLSClientConfig: tlsConfig,
				},
			}

			resp, err := c.Get(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Error(resp.StatusCode)
			}
		})
	}
}
