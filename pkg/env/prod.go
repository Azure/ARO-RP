package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2015-04-08/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	keyvaultmgmt "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type prod struct {
	instancemetadata.InstanceMetadata
	clientauthorizer.ClientAuthorizer

	keyvault keyvault.BaseClient

	dns dns.Manager

	cosmosDBAccountName      string
	cosmosDBPrimaryMasterKey string
	vaultURI                 string

	fpCertificate *x509.Certificate
	fpPrivateKey  *rsa.PrivateKey
}

func newProd(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata, clientauthorizer clientauthorizer.ClientAuthorizer) (*prod, error) {
	for _, key := range []string{
		"PULL_SECRET",
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

	p.keyvault.Authorizer, err = auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	err = p.populateCosmosDB(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateVaultURI(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	p.dns, err = dns.NewManager(ctx, instancemetadata, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	fpPrivateKey, fpCertificates, err := p.GetSecret(ctx, "rp-firstparty")
	if err != nil {
		return nil, err
	}

	p.fpPrivateKey = fpPrivateKey
	p.fpCertificate = fpCertificates[0]

	return p, nil
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

func (p *prod) CosmosDB() (string, string) {
	return p.cosmosDBAccountName, p.cosmosDBPrimaryMasterKey
}

func (p *prod) DatabaseName() string {
	return "ARO"
}

func (p *prod) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, network, address)
}

func (p *prod) DNS() dns.Manager {
	return p.dns
}

func (p *prod) FPAuthorizer(tenantID, resource string) (autorest.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875", p.fpCertificate, p.fpPrivateKey, resource)
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
