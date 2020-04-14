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
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
	"github.com/Azure/ARO-RP/test/util/listener"
)

func TestSecurity(t *testing.T) {
	ctx := context.Background()

	validclientkey, validclientcerts, err := utiltls.GenerateKeyAndCertificate("validclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	validadminclientkey, validadminclientcerts, err := utiltls.GenerateKeyAndCertificate("validclient", nil, nil, false, true)
	if err != nil {
		t.Fatal(err)
	}

	l := listener.NewListener()
	defer l.Close()

	env := &env.Test{
		L: l,
	}
	env.SetARMClientAuthorizer(clientauthorizer.NewOne(validclientcerts[0].Raw))
	env.SetAdminClientAuthorizer(clientauthorizer.NewOne(validadminclientcerts[0].Raw))

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

	f, err := NewFrontend(ctx, logrus.NewEntry(logrus.StandardLogger()), env, nil, api.APIs, &noop.Noop{}, nil, nil, nil)
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
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "admin operations url, no client certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
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
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            invalidclientkey,
			cert:           invalidclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "admin operations url, invalid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
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
			name:           "empty url, valid admin certificate",
			url:            "https://server/",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "unknown url, valid certificate",
			url:            "https://server/unknown",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "unknown url, valid admin certificate",
			url:            "https://server/unknown",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "operations url, valid certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "operations url, valid admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=2020-04-30",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "admin operations url, valid admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "admin operations url, valid non-admin certificate",
			url:            "https://server/providers/Microsoft.RedHatOpenShift/operations?api-version=admin",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ready url, valid certificate",
			url:            "https://server/healthz/ready",
			key:            validclientkey,
			cert:           validclientcerts[0],
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "ready url, valid admin certificate",
			url:            "https://server/healthz/ready",
			key:            validadminclientkey,
			cert:           validadminclientcerts[0],
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
