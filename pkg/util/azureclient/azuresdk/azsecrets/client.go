package azsecrets

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Client interface {
	BackupSecret(ctx context.Context, name string, options *azsecrets.BackupSecretOptions) (azsecrets.BackupSecretResponse, error)
	DeleteSecret(ctx context.Context, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error)
	GetDeletedSecret(ctx context.Context, name string, options *azsecrets.GetDeletedSecretOptions) (azsecrets.GetDeletedSecretResponse, error)
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
	NewListDeletedSecretPropertiesPager(options *azsecrets.ListDeletedSecretPropertiesOptions) *runtime.Pager[azsecrets.ListDeletedSecretPropertiesResponse]
	NewListSecretPropertiesPager(options *azsecrets.ListSecretPropertiesOptions) *runtime.Pager[azsecrets.ListSecretPropertiesResponse]
	NewListSecretPropertiesVersionsPager(name string, options *azsecrets.ListSecretPropertiesVersionsOptions) *runtime.Pager[azsecrets.ListSecretPropertiesVersionsResponse]
	PurgeDeletedSecret(ctx context.Context, name string, options *azsecrets.PurgeDeletedSecretOptions) (azsecrets.PurgeDeletedSecretResponse, error)
	RecoverDeletedSecret(ctx context.Context, name string, options *azsecrets.RecoverDeletedSecretOptions) (azsecrets.RecoverDeletedSecretResponse, error)
	RestoreSecret(ctx context.Context, parameters azsecrets.RestoreSecretParameters, options *azsecrets.RestoreSecretOptions) (azsecrets.RestoreSecretResponse, error)
	SetSecret(ctx context.Context, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
	UpdateSecretProperties(ctx context.Context, name string, version string, parameters azsecrets.UpdateSecretPropertiesParameters, options *azsecrets.UpdateSecretPropertiesOptions) (azsecrets.UpdateSecretPropertiesResponse, error)
}

type ArmClient struct {
	*azsecrets.Client
}

var _ Client = &ArmClient{}

func NewClient(vaultURL string, credential azcore.TokenCredential, options azcore.ClientOptions) (ArmClient, error) {
	azSecretsOptions := azsecrets.ClientOptions{
		ClientOptions: options,
	}
	_client, err := azsecrets.NewClient(vaultURL, credential, &azSecretsOptions)
	return ArmClient{
		Client: _client,
	}, err
}
