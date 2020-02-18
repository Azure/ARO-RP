package tls

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func CertAsBytes(certs ...*x509.Certificate) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}
	for _, cert := range certs {
		err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func CertChainAsBytes(certs []*x509.Certificate) (b []byte, err error) {
	return CertAsBytes(certs...)
}

func PrivateKeyAsBytes(key *rsa.PrivateKey) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}

	err = pem.Encode(buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
