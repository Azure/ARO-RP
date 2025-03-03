package tls

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

func GenerateKeyAndCertificate(commonName string, parentKey *rsa.PrivateKey, parentCert *x509.Certificate, isCA bool, isClient bool) (*rsa.PrivateKey, []*x509.Certificate, error) {
	return generateKeyAndCertificate(commonName, parentKey, parentCert, isCA, isClient, nil)
}

func MarshalKeyAndCertificate(key *rsa.PrivateKey, certs []*x509.Certificate) ([]byte, error) {
	b := bytes.Buffer{}
	b.Write(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	for _, cert := range certs {
		if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return []byte{}, err
		}
	}
	return b.Bytes(), nil
}

func GenerateTestKeyAndCertificate(commonName string, parentKey *rsa.PrivateKey, parentCert *x509.Certificate, isCA bool, isClient bool, tweakTemplate func(*x509.Certificate)) (*rsa.PrivateKey, []*x509.Certificate, error) {
	return generateKeyAndCertificate(commonName, parentKey, parentCert, isCA, isClient, tweakTemplate)
}

func generateKeyAndCertificate(commonName string, parentKey *rsa.PrivateKey, parentCert *x509.Certificate, isCA bool, isClient bool, tweakTemplate func(*x509.Certificate)) (*rsa.PrivateKey, []*x509.Certificate, error) {
	if isCA && isClient {
		return nil, nil, fmt.Errorf("cannot generate CA client certificate")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	notAfter := now.AddDate(1, 0, 0)

	if parentCert != nil && parentCert.NotAfter.Before(notAfter) {
		notAfter = parentCert.NotAfter
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
		NotAfter:              notAfter,
		Subject:               pkix.Name{CommonName: commonName},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		IsCA:                  isCA,
		DNSNames:              []string{commonName},
	}

	if tweakTemplate != nil {
		tweakTemplate(template)
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	} else {
		if isClient {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		} else {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		}
	}

	if parentCert == nil && parentKey == nil {
		parentCert = template
		parentKey = key
	}

	b, err := x509.CreateCertificate(rand.Reader, template, parentCert, &key.PublicKey, parentKey)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, nil, err
	}

	return key, []*x509.Certificate{cert}, nil
}
