package env

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type Env struct {
	databaseaccounts documentdb.DatabaseAccountsClient
	keyvault         keyvault.BaseClient
	vaults           keyvaultmgmt.VaultsClient
	zones            dns.ZonesClient

	resourceGroup string
}

func NewEnv(subscriptionId, resourceGroup string) (*Env, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	vaultauthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		return nil, err
	}

	e := &Env{
		resourceGroup: resourceGroup,
	}

	e.databaseaccounts = documentdb.NewDatabaseAccountsClient(subscriptionId)
	e.keyvault = keyvault.New()
	e.vaults = keyvaultmgmt.NewVaultsClient(subscriptionId)
	e.zones = dns.NewZonesClient(subscriptionId)

	e.databaseaccounts.Authorizer = authorizer
	e.keyvault.Authorizer = vaultauthorizer
	e.vaults.Authorizer = authorizer
	e.zones.Authorizer = authorizer

	return e, nil
}

func (e *Env) CosmosDB(ctx context.Context) (string, string, error) {
	accts, err := e.databaseaccounts.ListByResourceGroup(ctx, e.resourceGroup)
	if err != nil {
		return "", "", err
	}

	if len(*accts.Value) != 1 {
		return "", "", fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
	}

	keys, err := e.databaseaccounts.ListKeys(ctx, e.resourceGroup, *(*accts.Value)[0].Name)
	if err != nil {
		return "", "", err
	}

	return *(*accts.Value)[0].Name, *keys.PrimaryMasterKey, nil
}

func (e *Env) DNS(ctx context.Context) (string, error) {
	page, err := e.zones.ListByResourceGroup(ctx, e.resourceGroup, nil)
	if err != nil {
		return "", err
	}

	zones := page.Values()
	if len(zones) != 1 {
		return "", fmt.Errorf("found at least %d zones, expected 1", len(zones))
	}

	return *zones[0].Name, nil
}

func (e *Env) getSecret(ctx context.Context, vaultBaseURL, secretName string) (*rsa.PrivateKey, *x509.Certificate, error) {
	bundle, err := e.keyvault.GetSecret(ctx, vaultBaseURL, secretName, "")
	if err != nil {
		return nil, nil, err
	}

	var key *rsa.PrivateKey
	var cert *x509.Certificate
	b := []byte(*bundle.Value)
	for {
		var block *pem.Block
		block, b = pem.Decode(b)
		if block == nil {
			break
		}

		switch block.Type {
		case "PRIVATE KEY":
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			var ok bool
			key, ok = k.(*rsa.PrivateKey)
			if !ok {
				return nil, nil, errors.New("found unknown private key type in PKCS#8 wrapping")
			}

		case "CERTIFICATE":
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return key, cert, nil
}

func (e *Env) FirstPartyAuthorizer(ctx context.Context) (autorest.Authorizer, error) {
	page, err := e.vaults.ListByResourceGroup(ctx, e.resourceGroup, nil)
	if err != nil {
		return nil, err
	}

	vaults := page.Values()
	if len(vaults) != 1 {
		return nil, fmt.Errorf("found at least %d vaults, expected 1", len(vaults))
	}

	key, cert, err := e.getSecret(ctx, *vaults[0].Properties.VaultURI, "azure")
	if err != nil {
		return nil, err
	}

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, os.Getenv("AZURE_TENANT_ID"))
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, os.Getenv("AZURE_CLIENT_ID"), cert, key, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return autorest.NewBearerAuthorizer(sp), nil
}

func (e *Env) ServingCert(ctx context.Context) (*tls.Certificate, error) {
	page, err := e.vaults.ListByResourceGroup(ctx, e.resourceGroup, nil)
	if err != nil {
		return nil, err
	}

	vaults := page.Values()
	if len(vaults) != 1 {
		return nil, fmt.Errorf("found at least %d vaults, expected 1", len(vaults))
	}

	key, cert, err := e.getSecret(ctx, *vaults[0].Properties.VaultURI, "tls")
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{
			cert.Raw,
		},
		PrivateKey: key,
	}, nil
}
