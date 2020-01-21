package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
)

type Test struct {
	*prod

	L net.Listener

	TestSubscriptionID string
	TestLocation       string
	TestResourceGroup  string
	TestDomain         string
	TestVNetName       string
	TestSubnetName     string

	TLSKey   *rsa.PrivateKey
	TLSCerts []*x509.Certificate
}

func (t *Test) SetClientAuthorizer(clientauthorizer clientauthorizer.ClientAuthorizer) {
	if t.prod == nil {
		t.prod = &prod{}
	}
	t.ClientAuthorizer = clientauthorizer
}

func (t *Test) Domain() string {
	return t.TestDomain
}

func (t *Test) FPAuthorizer(tenantID, resource string) (autorest.Authorizer, error) {
	return nil, nil
}

func (t *Test) GetSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	switch secretName {
	case "rp-server":
		return t.TLSKey, t.TLSCerts, nil
	default:
		return nil, nil, fmt.Errorf("secret %q not found", secretName)
	}
}

func (t *Test) Listen() (net.Listener, error) {
	return t.L, nil
}

func (t *Test) Location() string {
	return t.TestLocation
}

func (t *Test) ResourceGroup() string {
	return t.TestResourceGroup
}

func (t *Test) SubnetName() string {
	return t.TestSubnetName
}

func (t *Test) SubscriptionID() string {
	return t.TestSubscriptionID
}

func (t *Test) VnetName() string {
	return t.TestVNetName
}
