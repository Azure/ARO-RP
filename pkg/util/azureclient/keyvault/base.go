package keyvault

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/azureclient/$GOPACKAGE/$GOPACKAGE.go github.com/Azure/ARO-RP/pkg/util/azureclient/$GOPACKAGE BaseClient
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/Azure/ARO-RP -e -w ../../../util/mocks/azureclient/$GOPACKAGE/$GOPACKAGE.go

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
