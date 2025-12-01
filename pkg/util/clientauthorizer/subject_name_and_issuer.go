package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/sirupsen/logrus"
)

// NewSubjectNameAndIssuer creates a new instance of ClientAuthorizer which
// allows connections only if they contain a valid client certificate signed by
// a CA in `certPool` and the client certificate's CommonName equals `clientCertCommonName`.
func NewSubjectNameAndIssuer(log *logrus.Entry, certPool *x509.CertPool, clientCertCommonName string) (ClientAuthorizer, error) {
	if clientCertCommonName == "" {
		return nil, fmt.Errorf("client cert common name is empty")
	}

	authorizer := &subjectNameAndIssuer{
		roots:                certPool,
		clientCertCommonName: clientCertCommonName,

		log: log,
	}

	return authorizer, nil
}

type subjectNameAndIssuer struct {
	roots                *x509.CertPool
	clientCertCommonName string

	log *logrus.Entry
}

func (sni *subjectNameAndIssuer) IsAuthorized(cs *tls.ConnectionState) bool {
	if sni.roots == nil {
		// Should never happen.  Do not fall back to system CA bundle.
		sni.log.Error("no CA certificate configured")
		return false
	}

	if cs == nil || len(cs.PeerCertificates) == 0 {
		sni.log.Debug("no certificate present for the connection")
		return false
	}

	verifyOpts := x509.VerifyOptions{
		Roots:         sni.roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, cert := range cs.PeerCertificates[1:] {
		verifyOpts.Intermediates.AddCert(cert)
	}

	_, err := cs.PeerCertificates[0].Verify(verifyOpts)
	if err != nil {
		sni.log.Debug(err)
		return false
	}

	if cs.PeerCertificates[0].Subject.CommonName != sni.clientCertCommonName {
		sni.log.Debug("unexpected common name in the admin API client certificate")
		return false
	}

	return true
}

func (sni *subjectNameAndIssuer) IsReady() bool {
	return true
}
