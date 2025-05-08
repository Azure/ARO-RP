package pem

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

type Encodable interface {
	*x509.Certificate | *x509.CertificateRequest | *rsa.PublicKey | *rsa.PrivateKey
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
		default:
			return nil, fmt.Errorf("unable to identify %v", t)
		}

		r = append(r, pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: rawBytes})...)
	}
	return
}
