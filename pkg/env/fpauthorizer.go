package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type FPAuthorizer interface {
	FPAuthorizer(string, string) (refreshable.Authorizer, error)
}

type fpAuthorizer struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
	spID string
}

func NewFPAuthorizer(ctx context.Context, env Lite, kv keyvault.Manager) (FPAuthorizer, error) {
	key, certs, err := kv.GetCertificateSecret(ctx, RPFirstPartySecretName)
	if err != nil {
		return nil, err
	}

	var spID string
	switch env.Type() {
	case Dev:
		for _, key := range []string{
			"AZURE_FP_CLIENT_ID",
		} {
			if _, found := os.LookupEnv(key); !found {
				return nil, fmt.Errorf("environment variable %q unset (development mode)", key)
			}
		}

		spID = os.Getenv("AZURE_FP_CLIENT_ID")
	case Int:
		spID = "71cfb175-ea3a-444e-8c03-b119b2752ce4"
	default:
		spID = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"
	}

	return &fpAuthorizer{
		key:  key,
		cert: certs[0],
		spID: spID,
	}, nil
}

func (m *fpAuthorizer) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, m.spID, m.cert, m.key, resource)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(sp), nil
}
