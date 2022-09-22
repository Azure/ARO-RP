package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"net/http"
)

type one struct {
	cert []byte
}

func NewOne(cert []byte) ClientAuthorizer {
	return &one{
		cert: cert,
	}
}

func (o *one) IsAuthorized(r *http.Request) bool {
	cs := r.TLS
	if cs == nil || len(cs.PeerCertificates) == 0 {
		return false
	}

	return bytes.Equal(o.cert, cs.PeerCertificates[0].Raw)
}

func (one) IsReady() bool {
	return true
}
