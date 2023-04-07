package pki

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
)

func TestNewPki(t *testing.T) {
	kpiUrl := "https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=%s"

	p := NewPki(kpiUrl)

	if p == nil {
		t.Error("Expected non-nil Pki instance")
	}
}

func TestGetTlsConfig(t *testing.T) {
	kpiUrl := "https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=%s"
	testUrl := "https://diag-runtimehost-prod.trafficmanager.net"

	p := NewPki(kpiUrl)

	caName := "ame"
	caCertPool, err := p.GetTlsCertPool(caName)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if caCertPool == nil {
		t.Error("Expected non-nil CertPool")
	}

	if _, ok := caMap[caName]; !ok {
		t.Errorf("Expected caMap to contain entry for %s", caName)
	}

	// Create a new tls.Config with the custom CA certificate
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	// Use the tls.Config with your client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	resp, err := client.Get(testUrl)

	if err != nil {
		if _, ok := err.(x509.UnknownAuthorityError); ok {
			t.Errorf("Invalid SSL/TLS certificate")
		}
		t.Errorf("Error calling %s", testUrl)
	}

	defer resp.Body.Close()
}
