package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	basekeyvault "github.com/Azure/ARO-RP/pkg/util/azureclient/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/dns"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/documentdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type prod struct {
	instancemetadata.InstanceMetadata
	armClientAuthorizer   clientauthorizer.ClientAuthorizer
	adminClientAuthorizer clientauthorizer.ClientAuthorizer

	keyvault basekeyvault.BaseClient

	acrName                  string
	clustersKeyvaultURI      string
	cosmosDBAccountName      string
	cosmosDBPrimaryMasterKey string
	domain                   string
	serviceKeyvaultURI       string
	zones                    map[string][]string

	fpCertificate        *x509.Certificate
	fpPrivateKey         *rsa.PrivateKey
	fpServicePrincipalID string

	clustersGenevaLoggingCertificate   *x509.Certificate
	clustersGenevaLoggingPrivateKey    *rsa.PrivateKey
	clustersGenevaLoggingConfigVersion string
	clustersGenevaLoggingEnvironment   string

	e2eStorageAccountName   string
	e2eStorageAccountRGName string
	e2eStorageAccountSubID  string

	log *logrus.Entry
}

func newProd(ctx context.Context, log *logrus.Entry, instancemetadata instancemetadata.InstanceMetadata, rpAuthorizer, kvAuthorizer autorest.Authorizer) (*prod, error) {
	p := &prod{
		InstanceMetadata: instancemetadata,

		keyvault: basekeyvault.New(kvAuthorizer),

		clustersGenevaLoggingEnvironment:   "DiagnosticsProd",
		clustersGenevaLoggingConfigVersion: "2.2",

		log: log,
	}

	err := p.populateCosmosDB(ctx, rpAuthorizer)
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

	err = p.populateZones(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	fpPrivateKey, fpCertificates, err := p.GetCertificateSecret(ctx, RPFirstPartySecretName)
	if err != nil {
		return nil, err
	}

	p.fpPrivateKey = fpPrivateKey
	p.fpCertificate = fpCertificates[0]
	p.fpServicePrincipalID = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"

	clustersGenevaLoggingPrivateKey, clustersGenevaLoggingCertificates, err := p.GetCertificateSecret(ctx, ClusterLoggingSecretName)
	if err != nil {
		return nil, err
	}

	p.clustersGenevaLoggingPrivateKey = clustersGenevaLoggingPrivateKey
	p.clustersGenevaLoggingCertificate = clustersGenevaLoggingCertificates[0]

	p.e2eStorageAccountName = "arov4e2e"
	p.e2eStorageAccountRGName = "global"
	p.e2eStorageAccountSubID = "0923c7de-9fca-4d9e-baf3-131d0c5b2ea4"

	if p.ACRResourceID() != "" { // TODO: ugh!
		acrResource, err := azure.ParseResourceID(p.ACRResourceID())
		if err != nil {
			return nil, err
		}
		p.acrName = acrResource.ResourceName
	}

	return p, nil
}

func (p *prod) InitializeAuthorizers() error {
	p.armClientAuthorizer = clientauthorizer.NewARM(p.log)

	adminClientAuthorizer, err := clientauthorizer.NewAdmin(
		p.log,
		"/etc/aro-rp/admin-ca-bundle.pem",
		os.Getenv("ADMIN_API_CLIENT_CERT_COMMON_NAME"),
	)
	if err != nil {
		return err
	}

	p.adminClientAuthorizer = adminClientAuthorizer
	return nil
}

func (p *prod) ArmClientAuthorizer() clientauthorizer.ClientAuthorizer {
	return p.armClientAuthorizer
}

func (p *prod) AdminClientAuthorizer() clientauthorizer.ClientAuthorizer {
	return p.adminClientAuthorizer
}

func (p *prod) ACRResourceID() string {
	return os.Getenv("ACR_RESOURCE_ID")
}

func (p *prod) ACRName() string {
	return p.acrName
}

func (p *prod) AROOperatorImage() string {
	return fmt.Sprintf("%s/aro:%s", p.acrName, version.GitCommit)
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
		if v.Tags[generator.KeyVaultTagName] != nil {
			switch *v.Tags[generator.KeyVaultTagName] {
			case generator.ClustersKeyVaultTagValue:
				p.clustersKeyvaultURI = *v.Properties.VaultURI
			case generator.ServiceKeyVaultTagValue:
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

func (p *prod) ClustersGenevaLoggingConfigVersion() string {
	return p.clustersGenevaLoggingConfigVersion
}

func (p *prod) ClustersGenevaLoggingEnvironment() string {
	return p.clustersGenevaLoggingEnvironment
}

func (p *prod) ClustersGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate) {
	return p.clustersGenevaLoggingPrivateKey, p.clustersGenevaLoggingCertificate
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

func (p *prod) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, p.fpServicePrincipalID, p.fpCertificate, p.fpPrivateKey, resource)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(sp), nil
}

func (p *prod) GetCertificateSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, []*x509.Certificate, error) {
	bundle, err := p.keyvault.GetSecret(ctx, p.serviceKeyvaultURI, secretName, "")
	if err != nil {
		return nil, nil, err
	}

	key, certs, err := pem.Parse([]byte(*bundle.Value))
	if err != nil {
		return nil, nil, err
	}

	if key == nil {
		return nil, nil, fmt.Errorf("no private key found")
	}

	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no certificate found")
	}

	return key, certs, nil
}

func (p *prod) GetSecret(ctx context.Context, secretName string) ([]byte, error) {
	bundle, err := p.keyvault.GetSecret(ctx, p.serviceKeyvaultURI, secretName, "")
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(*bundle.Value)
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

func (p *prod) MetricsSocketPath() string {
	return "/var/etw/mdm_statsd.socket"
}

func (p *prod) Zones(vmSize string) ([]string, error) {
	zones, found := p.zones[vmSize]
	if !found {
		return nil, fmt.Errorf("zone information not found for vm size %q", vmSize)
	}
	return zones, nil
}

func (p *prod) E2EStorageAccountName() string {
	return p.e2eStorageAccountName
}

func (p *prod) E2EStorageAccountRGName() string {
	return p.e2eStorageAccountRGName
}

func (p *prod) E2EStorageAccountSubID() string {
	return p.e2eStorageAccountSubID
}
