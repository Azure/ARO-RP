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
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/env/shared/dns"
)

type Shared struct {
	databaseaccounts documentdb.DatabaseAccountsClient
	keyvault         keyvault.BaseClient
	vaults           keyvaultmgmt.VaultsClient

	dns dns.Manager

	tenantID      string
	resourceGroup string
	vaultURI      string
}

func NewShared(ctx context.Context, log *logrus.Entry, tenantID, subscriptionID, resourceGroup string) (*Shared, error) {
	rpAuthorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	rpVaultAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		return nil, err
	}

	s := &Shared{
		tenantID:      tenantID,
		resourceGroup: resourceGroup,
	}

	s.databaseaccounts = documentdb.NewDatabaseAccountsClient(subscriptionID)
	s.keyvault = keyvault.New()
	s.vaults = keyvaultmgmt.NewVaultsClient(subscriptionID)

	s.databaseaccounts.Authorizer = rpAuthorizer
	s.keyvault.Authorizer = rpVaultAuthorizer
	s.vaults.Authorizer = rpAuthorizer

	page, err := s.vaults.ListByResourceGroup(ctx, s.resourceGroup, nil)
	if err != nil {
		return nil, err
	}

	vaults := page.Values()
	if len(vaults) != 1 {
		return nil, fmt.Errorf("found at least %d vaults, expected 1", len(vaults))
	}
	s.vaultURI = *vaults[0].Properties.VaultURI

	s.dns, err = dns.NewManager(ctx, subscriptionID, rpAuthorizer, s.resourceGroup)
	if err != nil {
		return nil, err
	}

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

func (s *Shared) DNS() dns.Manager {
	return s.dns
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

func (s *Shared) FPAuthorizer(ctx context.Context) (autorest.Authorizer, error) {
	key, cert, err := s.GetSecret(ctx, "azure")
	if err != nil {
		return nil, err
	}

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, s.tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, os.Getenv("AZURE_FP_CLIENT_ID"), cert, key, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return autorest.NewBearerAuthorizer(sp), nil
}
