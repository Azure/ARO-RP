package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

// BaseClient is a minimal interface for azure BaseClient
type BaseClient interface {
	CreateCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters azkeyvault.CertificateCreateParameters) (result azkeyvault.CertificateOperation, err error)
	DeleteCertificate(ctx context.Context, vaultBaseURL string, certificateName string) (result azkeyvault.DeletedCertificateBundle, err error)
	GetCertificateOperation(ctx context.Context, vaultBaseURL string, certificateName string) (result azkeyvault.CertificateOperation, err error)
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result azkeyvault.SecretBundle, err error)
	GetCertificates(ctx context.Context, vaultBaseURL string, maxresults *int32, includePending *bool) (result azkeyvault.CertificateListResultPage, err error)
	SetSecret(ctx context.Context, vaultBaseURL string, secretName string, parameters azkeyvault.SecretSetParameters) (result azkeyvault.SecretBundle, err error)
	BaseClientAddons
}

type baseClient struct {
	azkeyvault.BaseClient
}

var _ BaseClient = &baseClient{}

// New creates a new BaseClient
func New(authorizer autorest.Authorizer) BaseClient {
	client := azkeyvault.New()
	client.Authorizer = authorizer

	return &baseClient{
		BaseClient: client,
	}
}
