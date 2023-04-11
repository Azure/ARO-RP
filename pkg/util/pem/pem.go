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

type Encodable interface {
	*x509.Certificate | *x509.CertificateRequest | *rsa.PublicKey | *rsa.PrivateKey
}

func Parse(b []byte) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	for {
		var block *pem.Block
		block, b = pem.Decode(b)
		if block == nil {
			break
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, err
			}

		case "PRIVATE KEY":
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			var ok bool
			key, ok = k.(*rsa.PrivateKey)
			if !ok {
				return nil, nil, fmt.Errorf("unimplemented private key type %T in PKCS#8 wrapping", k)
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
	for {
		var block *pem.Block
		block, b = pem.Decode(b)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		return x509.ParseCertificate(block.Bytes)
	}

	return nil, errors.New("unable to find certificate")
}

func ParseFirstPrivateKey(b []byte) (*rsa.PrivateKey, error) {
	for {
		var block *pem.Block
		block, b = pem.Decode(b)
		if block == nil {
			break
		}

		if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
			continue
		}
		if block.Type == "RSA PRIVATE KEY" {
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		} else if block.Type == "PRIVATE KEY" {
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			var ok bool
			key, ok := k.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("unimplemented private key type %T in PKCS#8 wrapping", k)
			}
			return key, nil
		}
	}

	return nil, errors.New("unable to find key")
}

func Encode[V Encodable](inputs ...V) (r []byte, err error) {
	for _, i := range inputs {
		pemType := "UNKNOWN"
		var rawBytes []byte

		switch t := any(i).(type) {
		case *x509.Certificate:
			pemType = "CERTIFICATE"
			rawBytes = t.Raw
		case *x509.CertificateRequest:
			pemType = "CERTIFICATE REQUEST"
			rawBytes = t.Raw
		case *rsa.PrivateKey:
			pemType = "RSA PRIVATE KEY"
			rawBytes = x509.MarshalPKCS1PrivateKey(t)
		case *rsa.PublicKey:
			pemType = "RSA PUBLIC KEY"
			rawBytes, err = x509.MarshalPKIXPublicKey(t)
			if err != nil {
				return nil, err
			}
		}

		r = append(r, pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: rawBytes})...)
	}
	return
}
