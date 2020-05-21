package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type Test struct {
	*prod

	L net.Listener

	TestSubscriptionID string
	TestLocation       string
	TestResourceGroup  string
	TestDomain         string
	TestSecret         []byte

	TLSKey   *rsa.PrivateKey
	TLSCerts []*x509.Certificate
}

func (t *Test) SetARMClientAuthorizer(armClientAuthorizer clientauthorizer.ClientAuthorizer) {
	if t.prod == nil {
		t.prod = &prod{}
	}
	t.armClientAuthorizer = armClientAuthorizer
}

func (t *Test) SetAdminClientAuthorizer(adminClientAuthorizer clientauthorizer.ClientAuthorizer) {
	if t.prod == nil {
		t.prod = &prod{}
	}
	t.adminClientAuthorizer = adminClientAuthorizer
}

func (t *Test) Domain() string {
	return t.TestDomain
}

func (t *Test) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	return nil, nil
}

func (t *Test) GetCertificateSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	switch secretName {
	case RPServerSecretName:
		return t.TLSKey, t.TLSCerts, nil
	default:
		return nil, nil, fmt.Errorf("secret %q not found", secretName)
	}
}

func (t *Test) GetSecret(ctx context.Context, secretName string) ([]byte, error) {
	return t.TestSecret, nil
}

func (t *Test) Listen() (net.Listener, error) {
	return t.L, nil
}

func (t *Test) Location() string {
	return t.TestLocation
}

func (t *Test) ManagedDomain(clusterDomain string) (string, error) {
	if t.prod == nil {
		t.prod = &prod{}
	}
	t.prod.domain = t.TestDomain
	return t.prod.ManagedDomain(clusterDomain)
}

func (t *Test) ResourceGroup() string {
	return t.TestResourceGroup
}

func (t *Test) SubscriptionID() string {
	return t.TestSubscriptionID
}

func (t *Test) ACRResourceID() string {
	return "/subscriptions/93aeba23-2f76-4307-be82-02921df010cf/resourceGroups/global/providers/Microsoft.ContainerRegistry/registries/arointsvc"
}

func (t *Test) ACRName() string {
	return "arointsvc"
}
