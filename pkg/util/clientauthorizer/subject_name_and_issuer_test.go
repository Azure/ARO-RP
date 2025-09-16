package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/sirupsen/logrus"

	utiltls "github.com/Azure/ARO-RP/pkg/util/tls"
)

func TestSubjectNameAndIssuer(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	validCaKey, validCaCerts, err := utiltls.GenerateKeyAndCertificate("validca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	for _, cert := range validCaCerts {
		caCertPool.AddCert(cert)
	}

	for _, tt := range []struct {
		name string
		cs   func() (*tls.ConnectionState, error)
		want bool
	}{
		{
			name: "allow: single valid client certificate",
			want: true,
			cs: func() (*tls.ConnectionState, error) {
				_, validSingleClientCert, err := utiltls.GenerateKeyAndCertificate("validclient", validCaKey, validCaCerts[0], false, true)
				if err != nil {
					return nil, err
				}

				return &tls.ConnectionState{
					PeerCertificates: validSingleClientCert,
				}, nil
			},
		},
		{
			name: "allow: valid client certificate with intermediates",
			want: true,
			cs: func() (*tls.ConnectionState, error) {
				validIntermediateCaKey, validIntermediateCaCerts, err := utiltls.GenerateKeyAndCertificate("valid-intermediate-ca", validCaKey, validCaCerts[0], true, false)
				if err != nil {
					return nil, err
				}

				_, validCertWithIntermediates, err := utiltls.GenerateKeyAndCertificate("validclient", validIntermediateCaKey, validIntermediateCaCerts[0], false, true)
				if err != nil {
					return nil, err
				}
				validCertWithIntermediates = append(validCertWithIntermediates, validIntermediateCaCerts...)

				return &tls.ConnectionState{
					PeerCertificates: validCertWithIntermediates,
				}, nil
			},
		},
		{
			name: "deny: valid certificate with unexpected common name",
			cs: func() (*tls.ConnectionState, error) {
				_, invalidCommonNameClientCert, err := utiltls.GenerateKeyAndCertificate("invalidclient", validCaKey, validCaCerts[0], false, true)
				if err != nil {
					return nil, err
				}

				return &tls.ConnectionState{
					PeerCertificates: invalidCommonNameClientCert,
				}, nil
			},
		},
		{
			name: "deny: certificate with unexpected key usage",
			cs: func() (*tls.ConnectionState, error) {
				_, invalidKeyUsagesCert, err := utiltls.GenerateKeyAndCertificate("validclient", validCaKey, validCaCerts[0], false, false)
				if err != nil {
					return nil, err
				}

				return &tls.ConnectionState{
					PeerCertificates: invalidKeyUsagesCert,
				}, nil
			},
		},
		{
			name: "deny: matching common name, but unexpected ca",
			cs: func() (*tls.ConnectionState, error) {
				invalidCaKey, invalidCaCerts, err := utiltls.GenerateKeyAndCertificate("invalidca", nil, nil, true, false)
				if err != nil {
					return nil, err
				}

				_, invalidSigningCa, err := utiltls.GenerateKeyAndCertificate("validclient", invalidCaKey, invalidCaCerts[0], false, true)
				if err != nil {
					return nil, err
				}

				return &tls.ConnectionState{
					PeerCertificates: invalidSigningCa,
				}, nil
			},
		},
		{
			name: "deny: connection without client certificates",
			cs: func() (*tls.ConnectionState, error) {
				return &tls.ConnectionState{}, nil
			},
		},
		{
			name: "deny: nil connection state",
			cs: func() (*tls.ConnectionState, error) {
				return nil, nil
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authorizer, err := NewSubjectNameAndIssuer(log, caCertPool, "validclient")
			if err != nil {
				t.Fatal(err)
			}

			cs, err := tt.cs()
			if err != nil {
				t.Error(err)
			}

			result := authorizer.IsAuthorized(cs)
			if result != tt.want {
				t.Error(result)
			}
		})
	}
}
