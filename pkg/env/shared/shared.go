package shared

import (
	"context"
	"crypto/rsa"
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
	"github.com/sirupsen/logrus"
)

type Shared struct {
	databaseaccounts documentdb.DatabaseAccountsClient
	keyvault         keyvault.BaseClient
	vaults           keyvaultmgmt.VaultsClient
	zones            dns.ZonesClient

	resourceGroup string
	vaultURI      string
}

func NewShared(ctx context.Context, log *logrus.Entry, subscriptionId, resourceGroup string) (*Shared, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	vaultauthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		return nil, err
	}

	s := &Shared{
		resourceGroup: resourceGroup,
	}

	s.databaseaccounts = documentdb.NewDatabaseAccountsClient(subscriptionId)
	s.keyvault = keyvault.New()
	s.vaults = keyvaultmgmt.NewVaultsClient(subscriptionId)
	s.zones = dns.NewZonesClient(subscriptionId)

	s.databaseaccounts.Authorizer = authorizer
	s.keyvault.Authorizer = vaultauthorizer
	s.vaults.Authorizer = authorizer
	s.zones.Authorizer = authorizer

	page, err := s.vaults.ListByResourceGroup(ctx, s.resourceGroup, nil)
	if err != nil {
		return nil, err
	}

	vaults := page.Values()
	if len(vaults) != 1 {
		return nil, fmt.Errorf("found at least %d vaults, expected 1", len(vaults))
	}
	s.vaultURI = *vaults[0].Properties.VaultURI

	return s, nil
}

func (s *Shared) CosmosDB(ctx context.Context) (string, string, error) {
	accts, err := s.databaseaccounts.ListByResourceGroup(ctx, s.resourceGroup)
	if err != nil {
		return "", "", err
	}

	if len(*accts.Value) != 1 {
		return "", "", fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
	}

	keys, err := s.databaseaccounts.ListKeys(ctx, s.resourceGroup, *(*accts.Value)[0].Name)
	if err != nil {
		return "", "", err
	}

	return *(*accts.Value)[0].Name, *keys.PrimaryMasterKey, nil
}

func (s *Shared) DNS(ctx context.Context) (string, error) {
	page, err := s.zones.ListByResourceGroup(ctx, s.resourceGroup, nil)
	if err != nil {
		return "", err
	}

	zones := page.Values()
	if len(zones) != 1 {
		return "", fmt.Errorf("found at least %d zones, expected 1", len(zones))
	}

	return *zones[0].Name, nil
}

func (s *Shared) GetSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, *x509.Certificate, error) {
	bundle, err := s.keyvault.GetSecret(ctx, s.vaultURI, secretName, "")
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

func (s *Shared) FirstPartyAuthorizer(ctx context.Context) (autorest.Authorizer, error) {
	key, cert, err := s.GetSecret(ctx, "azure")
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
