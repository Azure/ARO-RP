package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	keySize = 2048

	// OneDay sets the validity of a cert to 24 hours.
	OneDay = time.Hour * 24

	// OneYear sets the validity of a cert to 1 year.
	OneYear = OneDay * 365

	// TenYears sets the validity of a cert to 10 years.
	TenYears = OneYear * 10
)

// CertCfg contains all needed fields to configure a new certificate
type CertCfg struct {
	DNSNames     []string
	IPAddresses  []net.IP
	KeyUsages    x509.KeyUsage
	ExtKeyUsages []x509.ExtKeyUsage
	Subject      pkix.Name
	Validity     time.Duration
}

// PrivateKey generates an RSA Private key and returns the value
func PrivateKey() (*rsa.PrivateKey, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, errors.Wrap(err, "error generating RSA private key")
	}

	return rsaKey, nil
}

// SignedCertificate creates a new X.509 certificate based on a template.
func SignedCertificate(
	cfg *CertCfg,
	csr *x509.CertificateRequest,
	key *rsa.PrivateKey,
	caCert *x509.Certificate,
	caKey *rsa.PrivateKey,
) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		DNSNames:              csr.DNSNames,
		ExtKeyUsage:           cfg.ExtKeyUsages,
		IPAddresses:           csr.IPAddresses,
		KeyUsage:              cfg.KeyUsages,
		NotAfter:              time.Now().Add(cfg.Validity),
		NotBefore:             caCert.NotBefore,
		SerialNumber:          serial,
		Subject:               csr.Subject,
		Version:               3,
		BasicConstraintsValid: true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create x509 certificate")
	}
	return x509.ParseCertificate(certBytes)
}

// GenerateSignedCertificate generate a key and cert defined by CertCfg and signed by CA.
func GenerateSignedCertificate(caKey *rsa.PrivateKey, caCert *x509.Certificate, cfg *CertCfg) (*rsa.PrivateKey, *x509.Certificate, error) {
	// create a private key
	key, err := PrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate private key")
	}

	// create a CSR
	csrTmpl := x509.CertificateRequest{Subject: cfg.Subject, DNSNames: cfg.DNSNames, IPAddresses: cfg.IPAddresses}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTmpl, key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create certificate request")
	}

	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		logrus.Debugf("Failed to parse x509 certificate request: %s", err)
		return nil, nil, errors.Wrap(err, "error parsing x509 certificate request")
	}

	// create a cert
	cert, err := SignedCertificate(cfg, csr, key, caCert, caKey)
	if err != nil {
		logrus.Debugf("Failed to create a signed certificate: %s", err)
		return nil, nil, errors.Wrap(err, "failed to create a signed certificate")
	}
	return key, cert, nil
}
