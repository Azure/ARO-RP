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

type test struct {
	*prod

	l net.Listener

	TLSKey   *rsa.PrivateKey
	TLSCerts []*x509.Certificate
}

func NewTest(l net.Listener, cert []byte) *test {
	return &test{
		prod: &prod{
			ClientAuthorizer: clientauthorizer.NewOne(cert),
		},
		l: l,
	}
}

func (t *test) Domain() string {
	return "test"
}

func (t *test) FPAuthorizer(tenantID, resource string) (autorest.Authorizer, error) {
	return nil, nil
}

func (t *test) GetSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	switch secretName {
	case "rp-server":
		return t.TLSKey, t.TLSCerts, nil
	default:
		return nil, nil, fmt.Errorf("secret %q not found", secretName)
	}
}

func (t *test) Listen() (net.Listener, error) {
	return t.l, nil
}

func (t *test) Location() string {
	return "eastus"
}

func (t *test) ResourceGroup() string {
	return "rpResourcegroup"
}

func (t *test) SubnetName() string {
	return "rpSubnet"
}

func (t *test) SubscriptionID() string {
	return "rpSubscriptionId"
}

func (t *test) VnetName() string {
	return "rpVnet"
}
