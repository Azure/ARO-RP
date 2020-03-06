package clientauthorizer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// NewAdmin creates a new instance of ClientAuthorizer to be used with Admin API.
// This authorizer allows connections only if they
// contain a valid client certificate signed by `caBundlePath` and
// the client certificate's CommonName equals `clientCertCommonName`.
func NewAdmin(log *logrus.Entry, caBundlePath, clientCertCommonName string) (ClientAuthorizer, error) {
	if clientCertCommonName == "" {
		return nil, fmt.Errorf("client cert common name is empty")
	}

	authorizer := &admin{
		clientCertCommonName: clientCertCommonName,

		log:      log,
		readFile: ioutil.ReadFile,
	}

	err := authorizer.readCABundle(caBundlePath)
	if err != nil {
		return nil, err
	}

	return authorizer, nil
}

type admin struct {
	roots                *x509.CertPool
	clientCertCommonName string

	log      *logrus.Entry
	readFile func(filename string) ([]byte, error)
}

func (a *admin) readCABundle(caBundlePath string) error {
	caBundle, err := a.readFile(caBundlePath)
	if err != nil {
		return err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caBundle)
	if !ok {
		return fmt.Errorf("can not decode admin CA bundle from %s", caBundlePath)
	}

	a.roots = roots
	return nil
}

func (a *admin) IsAuthorized(cs *tls.ConnectionState) bool {
	if a.roots == nil {
		// Should never happen.  Do not fall back to system CA bundle.
		a.log.Error("no CA certificate configured")
		return false
	}

	if cs == nil || len(cs.PeerCertificates) == 0 {
		a.log.Debug("no certificate present for the connection")
		return false
	}

	verifyOpts := x509.VerifyOptions{
		Roots:         a.roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, cert := range cs.PeerCertificates[1:] {
		verifyOpts.Intermediates.AddCert(cert)
	}

	_, err := cs.PeerCertificates[0].Verify(verifyOpts)
	if err != nil {
		a.log.Debug(err)
		return false
	}

	if cs.PeerCertificates[0].Subject.CommonName != a.clientCertCommonName {
		a.log.Debug("unexpected common name in the admin API client certificate")
		return false
	}

	return true
}

func (a *admin) IsReady() bool {
	return true
}
