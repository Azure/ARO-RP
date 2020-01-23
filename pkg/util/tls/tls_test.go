package tls

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"
	"testing"
)

func TestGenerateKeyAndCertificate(t *testing.T) {
	caKey, caCerts, err := GenerateKeyAndCertificate("ca", nil, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		parentKey  *rsa.PrivateKey
		parentCert *x509.Certificate
		isCA       bool
		isClient   bool
		wantErr    string
	}{
		{
			name: "self-signed server",
		},
		{
			name:     "self-signed client",
			isClient: true,
		},
		{
			name: "ca",
			isCA: true,
		},
		{
			name:     "invalid",
			isCA:     true,
			isClient: true,
			wantErr:  "cannot generate CA client certificate",
		},
		{
			name:       "ca-signed server",
			parentKey:  caKey,
			parentCert: caCerts[0],
		},
		{
			name:       "ca-signed client",
			parentKey:  caKey,
			parentCert: caCerts[0],
			isClient:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key, certs, err := GenerateKeyAndCertificate(tt.name, tt.parentKey, tt.parentCert, tt.isCA, tt.isClient)
			if err != nil {
				if err.Error() == tt.wantErr {
					return
				}
				t.Fatal(err)
			}

			if key == nil {
				t.Fatal(key)
			}

			if key.N.BitLen() != 2048 {
				t.Error(key.N.BitLen())
			}

			if len(certs) != 1 {
				t.Fatal(len(certs))
			}

			if tt.parentCert != nil {
				err = certs[0].CheckSignatureFrom(tt.parentCert)
			} else {
				err = certs[0].CheckSignature(certs[0].SignatureAlgorithm, certs[0].RawTBSCertificate, certs[0].Signature)
			}
			if err != nil {
				t.Error(err)
			}

			if certs[0].Version != 3 {
				t.Error(certs[0].Version)
			}

			if tt.parentCert != nil {
				if certs[0].Issuer.String() != tt.parentCert.Subject.String() {
					t.Error(certs[0].Issuer)
				}
			}

			if (certs[0].Subject.String() != pkix.Name{CommonName: tt.name}.String()) {
				t.Error(certs[0].Subject)
			}

			if tt.parentCert != nil {
				if certs[0].NotAfter != tt.parentCert.NotAfter {
					t.Error(certs[0].NotAfter)
				}
			} else {
				if certs[0].NotBefore.AddDate(1, 0, 0) != certs[0].NotAfter {
					t.Error(certs[0].NotBefore, certs[0].NotAfter)
				}
			}

			if tt.isCA {
				if certs[0].KeyUsage != x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment|x509.KeyUsageCertSign {
					t.Error(certs[0].KeyUsage)
				}
			} else {
				if certs[0].KeyUsage != x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment {
					t.Error(certs[0].KeyUsage)
				}
			}

			if tt.isCA {
				if len(certs[0].ExtKeyUsage) != 0 {
					t.Error(certs[0].ExtKeyUsage)
				}
			} else {
				if tt.isClient {
					if !reflect.DeepEqual(certs[0].ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}) {
						t.Error(certs[0].ExtKeyUsage)
					}
				} else {
					if !reflect.DeepEqual(certs[0].ExtKeyUsage, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}) {
						t.Error(certs[0].ExtKeyUsage)
					}
				}
			}

			if !certs[0].BasicConstraintsValid {
				t.Error(certs[0].BasicConstraintsValid)
			}

			if certs[0].IsCA != tt.isCA {
				t.Error(certs[0].IsCA)
			}
		})
	}
}
