package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type prod struct {
	Core
	proxy.Dialer

	armClientAuthorizer   clientauthorizer.ClientAuthorizer
	adminClientAuthorizer clientauthorizer.ClientAuthorizer

	acrDomain string
	zones     map[string][]string

	fpCertificate *x509.Certificate
	fpPrivateKey  *rsa.PrivateKey
	fpClientID    string

	clustersKeyvault keyvault.Manager
	serviceKeyvault  keyvault.Manager

	clustersGenevaLoggingCertificate   *x509.Certificate
	clustersGenevaLoggingPrivateKey    *rsa.PrivateKey
	clustersGenevaLoggingConfigVersion string
	clustersGenevaLoggingEnvironment   string

	log *logrus.Entry
}

func newProd(ctx context.Context, log *logrus.Entry) (*prod, error) {
	for _, key := range []string{
		"DOMAIN_NAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	core, err := NewCore(ctx, log)
	if err != nil {
		return nil, err
	}

	dialer, err := proxy.NewDialer(core.DeploymentMode())
	if err != nil {
		return nil, err
	}

	p := &prod{
		Core:   core,
		Dialer: dialer,

		clustersGenevaLoggingEnvironment:   "DiagnosticsProd",
		clustersGenevaLoggingConfigVersion: "2.2",

		log: log,
	}

	rpAuthorizer, err := p.NewRPAuthorizer(p.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	rpKVAuthorizer, err := p.NewRPAuthorizer(p.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	clustersKeyvaultURI, err := keyvault.URI(p, ClustersKeyvaultSuffix)
	if err != nil {
		return nil, err
	}

	serviceKeyvaultURI, err := keyvault.URI(p, ServiceKeyvaultSuffix)
	if err != nil {
		return nil, err
	}

	p.clustersKeyvault = keyvault.NewManager(rpKVAuthorizer, clustersKeyvaultURI)
	p.serviceKeyvault = keyvault.NewManager(rpKVAuthorizer, serviceKeyvaultURI)

	err = p.populateZones(ctx, rpAuthorizer)
	if err != nil {
		return nil, err
	}

	fpPrivateKey, fpCertificates, err := p.serviceKeyvault.GetCertificateSecret(ctx, RPFirstPartySecretName)
	if err != nil {
		return nil, err
	}

	p.fpPrivateKey = fpPrivateKey
	p.fpCertificate = fpCertificates[0]
	p.fpClientID = "f1dd0a37-89c6-4e07-bcd1-ffd3d43d8875"

	clustersGenevaLoggingPrivateKey, clustersGenevaLoggingCertificates, err := p.serviceKeyvault.GetCertificateSecret(ctx, ClusterLoggingSecretName)
	if err != nil {
		return nil, err
	}

	p.clustersGenevaLoggingPrivateKey = clustersGenevaLoggingPrivateKey
	p.clustersGenevaLoggingCertificate = clustersGenevaLoggingCertificates[0]

	if p.ACRResourceID() != "" { // TODO: ugh!
		acrResource, err := azure.ParseResourceID(p.ACRResourceID())
		if err != nil {
			return nil, err
		}
		p.acrDomain = acrResource.ResourceName + "." + p.Environment().ContainerRegistryDNSSuffix
	} else {
		p.acrDomain = "arointsvc" + "." + p.Environment().ContainerRegistryDNSSuffix
	}

	return p, nil
}

func (p *prod) InitializeAuthorizers() error {
	p.armClientAuthorizer = clientauthorizer.NewARM(p.log, p.Core)

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

func (p *prod) ACRDomain() string {
	return p.acrDomain
}

func (p *prod) AROOperatorImage() string {
	return fmt.Sprintf("%s/aro:%s", p.acrDomain, version.GitCommit)
}

func (p *prod) populateZones(ctx context.Context, rpAuthorizer autorest.Authorizer) error {
	c := compute.NewResourceSkusClient(p.Environment(), p.SubscriptionID(), rpAuthorizer)

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

		if len(*sku.LocationInfo) == 0 { // happened in eastus2euap
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

func (p *prod) ClustersKeyvault() keyvault.Manager {
	return p.clustersKeyvault
}

func (p *prod) Domain() string {
	return os.Getenv("DOMAIN_NAME")
}

func (p *prod) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(p.Environment().ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, p.fpClientID, p.fpCertificate, p.fpPrivateKey, resource)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(sp), nil
}

func (p *prod) Listen() (net.Listener, error) {
	return net.Listen("tcp", ":8443")
}

func (p *prod) ServiceKeyvault() keyvault.Manager {
	return p.serviceKeyvault
}

func (p *prod) Zones(vmSize string) ([]string, error) {
	zones, found := p.zones[vmSize]
	if !found {
		return nil, fmt.Errorf("zone information not found for vm size %q", vmSize)
	}
	return zones, nil
}

func (d *prod) CreateARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer refreshable.Authorizer, resourceGroup string) error {
	// ARM ResourceGroup role assignments are not required in production.
	return nil
}
