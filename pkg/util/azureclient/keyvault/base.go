package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// BaseClient is a minimal interface for azure BaseClient
type BaseClient interface {
	CreateCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateCreateParameters) (result keyvault.CertificateOperation, err error)
	DeleteCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.DeletedCertificateBundle, err error)
	GetCertificateOperation(ctx context.Context, vaultBaseURL string, certificateName string) (result keyvault.CertificateOperation, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
	SetCertificateIssuer(ctx context.Context, vaultBaseURL string, issuerName string, parameter keyvault.CertificateIssuerSetParameters) (result keyvault.IssuerBundle, err error)
	SetSecret(ctx context.Context, vaultBaseURL string, secretName string, parameters keyvault.SecretSetParameters) (result keyvault.SecretBundle, err error)
	BaseClientAddons
}

type baseClient struct {
	keyvault.BaseClient
}

var _ BaseClient = &baseClient{}

// New creates a new BaseClient
func New(authorizer autorest.Authorizer) BaseClient {
	client := keyvault.New()
	client.Authorizer = authorizer

	return &baseClient{
		BaseClient: client,
	}
}
