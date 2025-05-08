package azcertificates

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Client interface {
	BackupCertificate(ctx context.Context, name string, options *azcertificates.BackupCertificateOptions) (azcertificates.BackupCertificateResponse, error)
	CreateCertificate(ctx context.Context, name string, parameters azcertificates.CreateCertificateParameters, options *azcertificates.CreateCertificateOptions) (azcertificates.CreateCertificateResponse, error)
	DeleteCertificate(ctx context.Context, name string, options *azcertificates.DeleteCertificateOptions) (azcertificates.DeleteCertificateResponse, error)
	DeleteCertificateOperation(ctx context.Context, name string, options *azcertificates.DeleteCertificateOperationOptions) (azcertificates.DeleteCertificateOperationResponse, error)
	DeleteContacts(ctx context.Context, options *azcertificates.DeleteContactsOptions) (azcertificates.DeleteContactsResponse, error)
	DeleteIssuer(ctx context.Context, issuerName string, options *azcertificates.DeleteIssuerOptions) (azcertificates.DeleteIssuerResponse, error)
	GetCertificate(ctx context.Context, name string, version string, options *azcertificates.GetCertificateOptions) (azcertificates.GetCertificateResponse, error)
	GetCertificateOperation(ctx context.Context, name string, options *azcertificates.GetCertificateOperationOptions) (azcertificates.GetCertificateOperationResponse, error)
	GetCertificatePolicy(ctx context.Context, name string, options *azcertificates.GetCertificatePolicyOptions) (azcertificates.GetCertificatePolicyResponse, error)
	GetContacts(ctx context.Context, options *azcertificates.GetContactsOptions) (azcertificates.GetContactsResponse, error)
	GetDeletedCertificate(ctx context.Context, name string, options *azcertificates.GetDeletedCertificateOptions) (azcertificates.GetDeletedCertificateResponse, error)
	GetIssuer(ctx context.Context, issuerName string, options *azcertificates.GetIssuerOptions) (azcertificates.GetIssuerResponse, error)
	ImportCertificate(ctx context.Context, name string, parameters azcertificates.ImportCertificateParameters, options *azcertificates.ImportCertificateOptions) (azcertificates.ImportCertificateResponse, error)
	NewListCertificatePropertiesPager(options *azcertificates.ListCertificatePropertiesOptions) *runtime.Pager[azcertificates.ListCertificatePropertiesResponse]
	NewListCertificatePropertiesVersionsPager(name string, options *azcertificates.ListCertificatePropertiesVersionsOptions) *runtime.Pager[azcertificates.ListCertificatePropertiesVersionsResponse]
	NewListDeletedCertificatePropertiesPager(options *azcertificates.ListDeletedCertificatePropertiesOptions) *runtime.Pager[azcertificates.ListDeletedCertificatePropertiesResponse]
	NewListIssuerPropertiesPager(options *azcertificates.ListIssuerPropertiesOptions) *runtime.Pager[azcertificates.ListIssuerPropertiesResponse]
	MergeCertificate(ctx context.Context, name string, parameters azcertificates.MergeCertificateParameters, options *azcertificates.MergeCertificateOptions) (azcertificates.MergeCertificateResponse, error)
	PurgeDeletedCertificate(ctx context.Context, name string, options *azcertificates.PurgeDeletedCertificateOptions) (azcertificates.PurgeDeletedCertificateResponse, error)
	RecoverDeletedCertificate(ctx context.Context, name string, options *azcertificates.RecoverDeletedCertificateOptions) (azcertificates.RecoverDeletedCertificateResponse, error)
	RestoreCertificate(ctx context.Context, parameters azcertificates.RestoreCertificateParameters, options *azcertificates.RestoreCertificateOptions) (azcertificates.RestoreCertificateResponse, error)
	SetContacts(ctx context.Context, contacts azcertificates.Contacts, options *azcertificates.SetContactsOptions) (azcertificates.SetContactsResponse, error)
	SetIssuer(ctx context.Context, issuerName string, parameter azcertificates.SetIssuerParameters, options *azcertificates.SetIssuerOptions) (azcertificates.SetIssuerResponse, error)
	UpdateCertificate(ctx context.Context, name string, version string, parameters azcertificates.UpdateCertificateParameters, options *azcertificates.UpdateCertificateOptions) (azcertificates.UpdateCertificateResponse, error)
	UpdateCertificateOperation(ctx context.Context, name string, certificateOperation azcertificates.UpdateCertificateOperationParameter, options *azcertificates.UpdateCertificateOperationOptions) (azcertificates.UpdateCertificateOperationResponse, error)
	UpdateCertificatePolicy(ctx context.Context, name string, certificatePolicy azcertificates.CertificatePolicy, options *azcertificates.UpdateCertificatePolicyOptions) (azcertificates.UpdateCertificatePolicyResponse, error)
	UpdateIssuer(ctx context.Context, issuerName string, parameter azcertificates.UpdateIssuerParameters, options *azcertificates.UpdateIssuerOptions) (azcertificates.UpdateIssuerResponse, error)
}

type ArmClient struct {
	*azcertificates.Client
}

var _ Client = &ArmClient{}

func NewClient(vaultURL string, credential azcore.TokenCredential, options azcore.ClientOptions) (ArmClient, error) {
	clientOptions := azcertificates.ClientOptions{
		ClientOptions: options,
	}
	_client, err := azcertificates.NewClient(vaultURL, credential, &clientOptions)

	return ArmClient{
		Client: _client,
	}, err
}
