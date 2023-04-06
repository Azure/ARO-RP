package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CertInterface contains cert.
// See openshift/installer/pkg/asset/tls/tls.go
type CertInterface interface {
	// Cert returns the certificate.
	Cert() []byte
}

// CertKeyInterface contains a private key and the associated cert.
// See openshift/installer/pkg/asset/tls/tls.go
type CertKeyInterface interface {
	CertInterface
	// Key returns the private key.
	Key() []byte
}

// CertKey contains the private key and the cert.
// See openshift/installer/pkg/asset/tls/certkey.go
type CertKey struct {
	CertRaw []byte
	KeyRaw  []byte
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
func GenerateSignedCertKey(
	cfg *CertCfg,
	parentCA CertKeyInterface,
) ([]byte, []byte, error) {
	var key *rsa.PrivateKey
	var crt *x509.Certificate
	var err error

	caKey, err := PemToPrivateKey(parentCA.Key())
	if err != nil {
		logrus.Debugf("Failed to parse RSA private key: %s", err)
		return nil, nil, errors.Wrap(err, "failed to parse rsa private key")
	}

	caCert, err := PemToCertificate(parentCA.Cert())
	if err != nil {
		logrus.Debugf("Failed to parse x509 certificate: %s", err)
		return nil, nil, errors.Wrap(err, "failed to parse x509 certificate")
	}

	key, crt, err = GenerateSignedCertificate(caKey, caCert, cfg)
	if err != nil {
		logrus.Debugf("Failed to generate signed cert/key pair: %s", err)
		return nil, nil, errors.Wrap(err, "failed to generate signed cert/key pair")
	}

	return PrivateKeyToPem(key), CertToPem(crt), nil
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
