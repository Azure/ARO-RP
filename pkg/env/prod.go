package env

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/util/clientauthorizer"
	"github.com/jim-minter/rp/pkg/util/dns"
	"github.com/jim-minter/rp/pkg/util/instancemetadata"
)

type prod struct {
	instancemetadata.InstanceMetadata
	clientauthorizer.ClientAuthorizer

	keyvault keyvault.BaseClient

	dns dns.Manager

	vaultURI                 string
	cosmosDBAccountName      string
	cosmosDBPrimaryMasterKey string
}

func newProd(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata, clientauthorizer clientauthorizer.ClientAuthorizer) (*prod, error) {
	for _, key := range []string{
		"AZURE_FP_CLIENT_ID",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	p := &prod{
		InstanceMetadata: instancemetadata,
		ClientAuthorizer: clientauthorizer,

		keyvault: keyvault.New(),
	}

	rpAuthorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	p.keyvault.Authorizer, err = auth.NewAuthorizerFromEnvironmentWithResource("https://vault.azure.net")
	if err != nil {
		return nil, err
	}

	err = p.populateVaultURI(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateCosmosDB(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	p.dns, err = dns.NewManager(ctx, instancemetadata, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) populateVaultURI(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	vaults := keyvaultmgmt.NewVaultsClient(p.SubscriptionID())
	vaults.Authorizer = rpAuthorizer

	page, err := vaults.ListByResourceGroup(ctx, p.ResourceGroup(), nil)
	if err != nil {
		return err
	}

	vs := page.Values()
	if len(vs) != 1 {
		return fmt.Errorf("found at least %d vaults, expected 1", len(vs))
	}

	p.vaultURI = *vs[0].Properties.VaultURI

	return nil
}

func (p *prod) populateCosmosDB(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	databaseaccounts := documentdb.NewDatabaseAccountsClient(p.SubscriptionID())
	databaseaccounts.Authorizer = rpAuthorizer

	accts, err := databaseaccounts.ListByResourceGroup(ctx, p.ResourceGroup())
	if err != nil {
		return err
	}

	if len(*accts.Value) != 1 {
		return fmt.Errorf("found %d database accounts, expected 1", len(*accts.Value))
	}

	keys, err := databaseaccounts.ListKeys(ctx, p.ResourceGroup(), *(*accts.Value)[0].Name)
	if err != nil {
		return err
	}

	p.cosmosDBAccountName = *(*accts.Value)[0].Name
	p.cosmosDBPrimaryMasterKey = *keys.PrimaryMasterKey

	return nil
}

func (p *prod) CosmosDB(context.Context) (string, string) {
	return p.cosmosDBAccountName, p.cosmosDBPrimaryMasterKey
}

func (p *prod) DNS() dns.Manager {
	return p.dns
}

func (p *prod) FPAuthorizer(ctx context.Context, resource string) (autorest.Authorizer, error) {
	sp, err := p.fpToken(ctx, resource)
	if err != nil {
		return nil, err
	}

	return autorest.NewBearerAuthorizer(sp), nil
}

func (p *prod) GetSecret(ctx context.Context, secretName string) (key *rsa.PrivateKey, certs []*x509.Certificate, err error) {
	bundle, err := p.keyvault.GetSecret(ctx, p.vaultURI, secretName, "")
	if err != nil {
		return nil, nil, err
	}

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
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			certs = append(certs, c)
		}
	}

	if key == nil {
		return nil, nil, errors.New("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, errors.New("no certificate found")
	}

	return key, certs, nil
}

func (p *prod) Listen() (net.Listener, error) {
	return net.Listen("tcp", ":8443")
}

func (p *prod) fpToken(ctx context.Context, resource string) (*adal.ServicePrincipalToken, error) {
	key, certs, err := p.GetSecret(ctx, "rp-firstparty")
	if err != nil {
		return nil, err
	}

	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, p.TenantID())
	if err != nil {
		return nil, err
	}

	return adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, os.Getenv("AZURE_FP_CLIENT_ID"), certs[0], key, resource)
}
