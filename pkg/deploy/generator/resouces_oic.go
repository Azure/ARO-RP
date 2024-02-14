package generator

import (
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (g *generator) oicStorageAccount() *arm.Resource {
	storageAccount := &mgmtstorage.Account{
		Kind: mgmtstorage.StorageV2,
		Sku: &mgmtstorage.Sku{
			Name: "Standard_LRS",
		},
		AccountProperties: &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess:  to.BoolPtr(true),
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
			MinimumTLSVersion:      mgmtstorage.TLS12,
			AccessTier:             mgmtstorage.Hot,
		},
		Name:     to.StringPtr("[concat(take(replace(resourceGroup().name, '-', ''), 21), 'oic')]"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
	}

	return &arm.Resource{
		Resource:   storageAccount,
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}
