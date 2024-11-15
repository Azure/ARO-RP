package store

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

//go:generate /bin/bash -c "../../hack/mockgen.sh mock_kvclient/zz_generated_mocks.go kvclient.go"

type KeyVaultClient interface {
	DeleteSecret(ctx context.Context, name string, options *azsecrets.DeleteSecretOptions) (azsecrets.DeleteSecretResponse, error)
	GetDeletedSecret(ctx context.Context, name string, options *azsecrets.GetDeletedSecretOptions) (azsecrets.GetDeletedSecretResponse, error)
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
	NewListDeletedSecretPropertiesPager(options *azsecrets.ListDeletedSecretPropertiesOptions) *runtime.Pager[azsecrets.ListDeletedSecretPropertiesResponse]
	NewListSecretPropertiesPager(options *azsecrets.ListSecretPropertiesOptions) *runtime.Pager[azsecrets.ListSecretPropertiesResponse]
	PurgeDeletedSecret(ctx context.Context, name string, options *azsecrets.PurgeDeletedSecretOptions) (azsecrets.PurgeDeletedSecretResponse, error)
	SetSecret(ctx context.Context, name string, parameters azsecrets.SetSecretParameters, options *azsecrets.SetSecretOptions) (azsecrets.SetSecretResponse, error)
}
