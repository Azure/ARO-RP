package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy"
	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/documentdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type prod struct {
	instancemetadata.InstanceMetadata
	clientauthorizer.ClientAuthorizer

	keyvault basekeyvault.BaseClient

	clustersKeyvaultURI      string
	cosmosDBAccountName      string
	cosmosDBPrimaryMasterKey string
	domain                   string
	serviceKeyvaultURI       string
	vnetName                 string
	zones                    map[string][]string

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

	kvAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	p := &prod{
		InstanceMetadata: instancemetadata,
		ClientAuthorizer: clientauthorizer,

		keyvault: basekeyvault.New(kvAuthorizer),
	}

	rpAuthorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	err = p.populateCosmosDB(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateDomain(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateVaultURIs(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateVnet(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	err = p.populateZones(ctx, rpAuthorizer)
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
	databaseaccounts := documentdb.NewDatabaseAccountsClient(p.SubscriptionID(), rpAuthorizer)

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

func (p *prod) populateDomain(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	zones := dns.NewZonesClient(p.SubscriptionID(), rpAuthorizer)

	zs, err := zones.ListByResourceGroup(ctx, p.ResourceGroup(), nil)
	if err != nil {
		return err
	}

	if len(zs) != 1 {
		return fmt.Errorf("found %d zones, expected 1", len(zs))
	}

	p.domain = *zs[0].Name

	return nil
}

func (p *prod) populateVaultURIs(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	vaults := keyvault.NewVaultsClient(p.SubscriptionID(), rpAuthorizer)

	vs, err := vaults.ListByResourceGroup(ctx, p.ResourceGroup(), nil)
	if err != nil {
		return err
	}

	for _, v := range vs {
		if v.Tags[deploy.KeyVaultTagName] != nil {
			switch *v.Tags[deploy.KeyVaultTagName] {
			case deploy.ClustersKeyVaultTagValue:
				p.clustersKeyvaultURI = *v.Properties.VaultURI
			case deploy.ServiceKeyVaultTagValue:
				p.serviceKeyvaultURI = *v.Properties.VaultURI
			}
		}
	}

	if p.clustersKeyvaultURI == "" {
		return fmt.Errorf("clusters key vault not found")
	}

	if p.serviceKeyvaultURI == "" {
		return fmt.Errorf("service key vault not found")
	}

	return nil
}

func (p *prod) populateVnet(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	virtualnetworks := network.NewVirtualNetworksClient(p.SubscriptionID(), rpAuthorizer)

	vnets, err := virtualnetworks.List(ctx, p.ResourceGroup())
	if err != nil {
		return err
	}

	for i := 0; i < len(vnets); {
		if vnets[i].Tags["vnet"] == nil || *vnets[i].Tags["vnet"] != "rp" {
			vnets = append(vnets[:i], vnets[i+1:]...)
		} else {
			i++
		}
	}

	if len(vnets) != 1 {
		return fmt.Errorf("found %d virtual networks, expected 1", len(vnets))
	}

	p.vnetName = *(vnets[0]).Name

	return nil
}

func (p *prod) populateZones(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	c := compute.NewResourceSkusClient(p.SubscriptionID(), rpAuthorizer)

	skus, err := c.List(ctx, "")
	if err != nil {
		return err
	}

	p.zones = map[string][]string{}

	for _, sku := range skus {
		if !strings.EqualFold((*sku.Locations)[0], p.Location()) ||
			*sku.ResourceType != "virtualMachines" {
			continue
		}

		p.zones[*sku.Name] = *(*sku.LocationInfo)[0].Zones
	}

	return nil
}

func (p *prod) ClustersKeyvaultURI() string {
	return p.clustersKeyvaultURI
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

func (p *prod) Domain() string {
	return p.domain
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
	bundle, err := p.keyvault.GetSecret(ctx, p.serviceKeyvaultURI, secretName, "")
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
				return nil, nil, fmt.Errorf("found unimplemented private key type %T in PKCS#8 wrapping", k)
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
		return nil, nil, fmt.Errorf("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate found")
	}

	return key, certs, nil
}

func (p *prod) Listen() (net.Listener, error) {
	return net.Listen("tcp", ":8443")
}

// ManagedDomain returns the fully qualified domain of a cluster if we manage
// it.  If we don't, it returns the empty string.  We manage only domains of the
// form "foo.$LOCATION.aroapp.io" and "foo" (we consider this a short form of
// the former).
func (p *prod) ManagedDomain(domain string) (string, error) {
	if domain == "" ||
		strings.HasPrefix(domain, ".") ||
		strings.HasSuffix(domain, ".") {
		// belt and braces: validation should already prevent this
		return "", fmt.Errorf("invalid domain %q", domain)
	}

	domain = strings.TrimSuffix(domain, "."+p.Domain())
	if strings.ContainsRune(domain, '.') {
		return "", nil
	}
	return domain + "." + p.Domain(), nil
}

func (p *prod) VnetName() string {
	return p.vnetName
}

func (p *prod) Zones(vmSize string) ([]string, error) {
	zones, found := p.zones[vmSize]
	if !found {
		return nil, fmt.Errorf("zone information not found for vm size %q", vmSize)
	}
	return zones, nil
}
