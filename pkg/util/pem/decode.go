package pem

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

func parsePrivateKey(block *pem.Block) (*rsa.PrivateKey, error) {
	var key *rsa.PrivateKey
	// try PKCS1
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		// the key is PKCS1, return it
		return key, nil
	}

	// if it's not PKCS1, try PKCS8
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("private key is not PKCS#1 or PKCS#8")
	}

	var ok bool
	key, ok = k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unimplemented private key type %T in PKCS#8 wrapping", k)
	}

	return key, nil
}

func Parse(b []byte) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	for {
		var block *pem.Block
		block, b = pem.Decode(b)
		if block == nil {
			break
		}

		switch block.Type {
		case "RSA PRIVATE KEY", "PRIVATE KEY":
			key, err = parsePrivateKey(block)
			if err != nil {
				return nil, nil, err
			}
		case "CERTIFICATE":
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			certs = append(certs, c)

		default:
			return nil, nil, fmt.Errorf("unimplemented block type %s", block.Type)
		}
	}

	return key, certs, nil
}

func ParseFirstCertificate(b []byte) (*x509.Certificate, error) {
	_, certs, err := Parse(b)
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 {
		return nil, errors.New("unable to find certificate")
	}

	return certs[0], nil
}

func ParseFirstPrivateKey(b []byte) (*rsa.PrivateKey, error) {
	key, _, err := Parse(b)
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, errors.New("unable to find key")
	}

	return key, nil
}
