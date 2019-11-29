package storage

//go:generate go run ../../../../vendor/github.com/golang/mock/mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/jim-minter/rp/pkg/util/azureclient/$GOPACKAGE AccountsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate go run ../../../../vendor/golang.org/x/tools/cmd/goimports -local=github.com/jim-minter/rp -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
)

// AccountsClient is a minimal interface for azure AccountsClient
type AccountsClient interface {
	ListKeys(ctx context.Context, resourceGroupName string, accountName string, expand storage.ListKeyExpand) (result storage.AccountListKeysResult, err error)
	ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error)
	AccountsClientAddons
}

func (a *accountsClient) Client() autorest.Client {
	return a.AccountsClient.Client
}

type accountsClient struct {
	storage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient returns a new AccountsClient
func NewAccountsClient(subscriptionID string, authorizer autorest.Authorizer) AccountsClient {
	client := storage.NewAccountsClient(subscriptionID)
	client.Authorizer = authorizer

	return &accountsClient{
		AccountsClient: client,
	}
}
