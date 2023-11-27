package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"

	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
)

// CertKeyInterface contains a private key and the associated cert.
// See openshift/installer/pkg/asset/tls/tls.go
type CertKeyInterface interface {
	// Cert returns the certificate.
	Cert() []byte
	// Key returns the private key.
	Key() []byte
}

// CertKey contains the private key and the cert.
// See openshift/installer/pkg/asset/tls/certkey.go
type CertKey struct {
	CertRaw []byte `json:"CertRaw"`
	KeyRaw  []byte `json:"KeyRaw"`
}

// Cert returns the certificate.
func (c *CertKey) Cert() []byte {
	return c.CertRaw
}

// Key returns the private key.
func (c *CertKey) Key() []byte {
	return c.KeyRaw
}

// SignedCertKey contains the private key and the cert that's
// signed by the parent CA.
type SignedCertKey struct {
	CertKey
}

// Generate generates a cert/key pair signed by the specified parent CA.
// see signedcertkey
func GenerateSignedCertKey(cfg *CertCfg, parentCA CertKeyInterface) (*rsa.PrivateKey, *x509.Certificate, error) {
	caKey, err := utilpem.ParseFirstPrivateKey(parentCA.Key())
	if err != nil {
		return nil, nil, err
	}

	cert, err := utilpem.ParseFirstCertificate(parentCA.Cert())
	if err != nil {
		return nil, nil, err
	}

	return GenerateSignedCertificate(caKey, cert, cfg)
}

// SelfSignedCertKey contains the private key and the cert that's self-signed.
type SelfSignedCertKey struct {
	CertKey
}

// RootCA contains the private key and the cert that's
// self-signed as the root CA.
type RootCA struct {
	SelfSignedCertKey
}
