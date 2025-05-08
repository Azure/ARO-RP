package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"crypto/tls"
)

type one struct {
	cert []byte
}

func NewOne(cert []byte) ClientAuthorizer {
	return &one{
		cert: cert,
	}
}

func (o *one) IsAuthorized(cs *tls.ConnectionState) bool {
	if cs == nil || len(cs.PeerCertificates) == 0 {
		return false
	}

	return bytes.Equal(o.cert, cs.PeerCertificates[0].Raw)
}

func (one) IsReady() bool {
	return true
}
