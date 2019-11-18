package env

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest"
)

func CosmosDB(ctx context.Context, authorizer autorest.Authorizer) (string, string, error) {
	dac := documentdb.NewDatabaseAccountsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"))
	dac.Authorizer = authorizer

	accts, err := dac.ListByResourceGroup(ctx, os.Getenv("RESOURCEGROUP"))
	if err != nil {
		return "", "", err
	}

	if len(*accts.Value) != 1 {
		return "", "", fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
	}

	keys, err := dac.ListKeys(ctx, os.Getenv("RESOURCEGROUP"), *(*accts.Value)[0].Name)
	if err != nil {
		return "", "", err
	}

	return *(*accts.Value)[0].Name, *keys.PrimaryMasterKey, nil
}

func DNS(ctx context.Context, authorizer autorest.Authorizer) (string, error) {
	zc := dns.NewZonesClient(os.Getenv("AZURE_SUBSCRIPTION_ID"))
	zc.Authorizer = authorizer

	page, err := zc.ListByResourceGroup(ctx, os.Getenv("RESOURCEGROUP"), nil)
	if err != nil {
		return "", err
	}
	zones := page.Values()
	if len(zones) != 1 {
		return "", fmt.Errorf("found at least %d zones, expected 1", len(zones))
	}

	return *zones[0].Name, nil
}
