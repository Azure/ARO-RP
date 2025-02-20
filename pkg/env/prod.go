package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/jongio/azidext/go/azidext"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/computeskus"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
	"github.com/Azure/ARO-RP/pkg/util/miseadapter"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	KeyvaultPrefix         = "KEYVAULT_PREFIX"
	OIDCAFDEndpoint        = "OIDC_AFD_ENDPOINT"
	OIDCStorageAccountName = "OIDC_STORAGE_ACCOUNT_NAME"
	OtelAuditQueueSize     = "OTEL_AUDIT_QUEUE_SIZE"
)

type prod struct {
	Core
	proxy.Dialer
	ARMHelper

	liveConfig liveconfig.Manager

	armClientAuthorizer   clientauthorizer.ClientAuthorizer
	adminClientAuthorizer clientauthorizer.ClientAuthorizer
	miseAuthorizer        miseadapter.MISEAdapter

	acrDomain string
	vmskus    map[string]*mgmtcompute.ResourceSku

	fpCertificateRefresher CertificateRefresher
	fpClientID             string

	clusterKeyvault keyvault.Manager
	serviceKeyvault keyvault.Manager

	clusterGenevaLoggingCertificate   *x509.Certificate
	clusterGenevaLoggingPrivateKey    *rsa.PrivateKey
	clusterGenevaLoggingAccount       string
	clusterGenevaLoggingConfigVersion string
	clusterGenevaLoggingEnvironment   string
	clusterGenevaLoggingNamespace     string

	gatewayDomains []string

	log *logrus.Entry

	features map[Feature]bool
}

func newProd(ctx context.Context, log *logrus.Entry, component ServiceComponent) (*prod, error) {
	if err := ValidateVars("AZURE_FP_CLIENT_ID", "DOMAIN_NAME"); err != nil {
		return nil, err
	}

	if !IsLocalDevelopmentMode() {
		err := ValidateVars(
			"CLUSTER_MDSD_CONFIG_VERSION",
			"CLUSTER_MDSD_ACCOUNT",
			"GATEWAY_DOMAINS",
			"GATEWAY_RESOURCEGROUP",
			"MDSD_ENVIRONMENT",
			"CLUSTER_MDSD_NAMESPACE")

		if err != nil {
			return nil, err
		}
	}

	core, err := NewCore(ctx, log, component)
	if err != nil {
		return nil, err
	}

	dialer, err := proxy.NewDialer(core.IsLocalDevelopmentMode())
	if err != nil {
		return nil, err
	}

	p := &prod{
		Core:   core,
		Dialer: dialer,

		fpClientID: os.Getenv("AZURE_FP_CLIENT_ID"),

		clusterGenevaLoggingAccount:       os.Getenv("CLUSTER_MDSD_ACCOUNT"),
		clusterGenevaLoggingConfigVersion: os.Getenv("CLUSTER_MDSD_CONFIG_VERSION"),
		clusterGenevaLoggingEnvironment:   os.Getenv("MDSD_ENVIRONMENT"),
		clusterGenevaLoggingNamespace:     os.Getenv("CLUSTER_MDSD_NAMESPACE"),

		log: log,

		features: map[Feature]bool{},
	}

	features := os.Getenv("RP_FEATURES")
	if features != "" {
		for _, feature := range strings.Split(features, ",") {
			f, err := FeatureString("Feature" + feature)
			if err != nil {
				return nil, err
			}

			p.features[f] = true
		}
	}

	msiAuthorizer, err := p.NewMSIAuthorizer(p.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	msiKVAuthorizer, err := p.NewMSIAuthorizer(p.Environment().KeyVaultScope)
	if err != nil {
		return nil, err
	}

	if err := ValidateVars(KeyvaultPrefix); err != nil {
		return nil, err
	}
	keyVaultPrefix := os.Getenv(KeyvaultPrefix)
	serviceKeyvaultURI := keyvault.URI(p, ServiceKeyvaultSuffix, keyVaultPrefix)
	p.serviceKeyvault = keyvault.NewManager(msiKVAuthorizer, serviceKeyvaultURI)

	resourceSkusClient := compute.NewResourceSkusClient(p.Environment(), p.SubscriptionID(), msiAuthorizer)
	err = p.populateVMSkus(ctx, resourceSkusClient)
	if err != nil {
		return nil, err
	}

	p.fpCertificateRefresher = newCertificateRefresher(log, 1*time.Hour, p.serviceKeyvault, RPFirstPartySecretName)
	err = p.fpCertificateRefresher.Start(ctx)
	if err != nil {
		return nil, err
	}

	localFPKVAuthorizer, err := p.FPAuthorizer(p.TenantID(), nil, p.Environment().KeyVaultScope)
	if err != nil {
		return nil, err
	}

	clusterKeyvaultURI := keyvault.URI(p, ClusterKeyvaultSuffix, keyVaultPrefix)
	p.clusterKeyvault = keyvault.NewManager(localFPKVAuthorizer, clusterKeyvaultURI)

	clusterGenevaLoggingPrivateKey, clusterGenevaLoggingCertificates, err := p.serviceKeyvault.GetCertificateSecret(ctx, ClusterLoggingSecretName)
	if err != nil {
		return nil, err
	}

	p.clusterGenevaLoggingPrivateKey = clusterGenevaLoggingPrivateKey
	p.clusterGenevaLoggingCertificate = clusterGenevaLoggingCertificates[0]

	var acrDataDomain string
	if p.ACRResourceID() != "" { // TODO: ugh!
		acrResource, err := azure.ParseResourceID(p.ACRResourceID())
		if err != nil {
			return nil, err
		}
		p.acrDomain = acrResource.ResourceName + "." + p.Environment().ContainerRegistryDNSSuffix
		acrDataDomain = acrResource.ResourceName + "." + p.Location() + ".data." + p.Environment().ContainerRegistryDNSSuffix
	} else {
		p.acrDomain = "arointsvc." + azure.PublicCloud.ContainerRegistryDNSSuffix                             // TODO: make cloud aware once this is set up for US Gov Cloud
		acrDataDomain = "arointsvc." + p.Location() + ".data." + azure.PublicCloud.ContainerRegistryDNSSuffix // TODO: make cloud aware once this is set up for US Gov Cloud
	}

	if !p.IsLocalDevelopmentMode() {
		gatewayDomains := os.Getenv("GATEWAY_DOMAINS")
		if gatewayDomains != "" {
			p.gatewayDomains = strings.Split(gatewayDomains, ",")
		}

		for _, rawurl := range []string{
			p.Environment().ActiveDirectoryEndpoint,
			p.Environment().ResourceManagerEndpoint,
		} {
			u, err := url.Parse(rawurl)
			if err != nil {
				return nil, err
			}

			p.gatewayDomains = append(p.gatewayDomains, u.Hostname())
		}

		p.gatewayDomains = append(p.gatewayDomains, p.acrDomain, acrDataDomain)
	}

	p.ARMHelper, err = newARMHelper(ctx, log, p)
	if err != nil {
		return nil, err
	}

	p.liveConfig, err = p.Core.NewLiveConfigManager(ctx)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *prod) MISEAuthorizer() miseadapter.MISEAdapter {
	return p.miseAuthorizer
}

func (p *prod) InitializeAuthorizers() error {
	if p.FeatureIsSet(FeatureEnableMISE) {
		err := ValidateVars(
			"MISE_ADDRESS",
		)
		if err != nil {
			return err
		}
		p.miseAuthorizer = miseadapter.NewAuthorizer(os.Getenv("MISE_ADDRESS"), p.log)
	}

	if !p.FeatureIsSet(FeatureEnforceMISE) {
		if !p.FeatureIsSet(FeatureEnableDevelopmentAuthorizer) {
			p.armClientAuthorizer = clientauthorizer.NewARM(p.log, p.Core)
		} else {
			armClientAuthorizer, err := clientauthorizer.NewSubjectNameAndIssuer(
				p.log,
				"/etc/aro-rp/arm-ca-bundle.pem",
				os.Getenv("ARM_API_CLIENT_CERT_COMMON_NAME"),
			)
			if err != nil {
				return err
			}

			p.armClientAuthorizer = armClientAuthorizer
		}
	}

	adminClientAuthorizer, err := clientauthorizer.NewSubjectNameAndIssuer(
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

func (p *prod) OIDCStorageAccountName() string {
	return os.Getenv(OIDCStorageAccountName)
}

func (p *prod) OIDCEndpoint() string {
	return fmt.Sprintf("https://%s/", os.Getenv("OIDC_AFD_ENDPOINT"))
}

func (p *prod) OIDCKeyBitSize() int {
	return 4096
}

// OtelAuditQueueSize returns the size of the otel audit queue.
// If the OTEL_AUDIT_QUEUE_SIZE environment variable is not set, it returns the default value of 4000.
func (p *prod) OtelAuditQueueSize() (int, error) {
	if err := ValidateVars(OtelAuditQueueSize); err != nil {
		return 4000, nil
	}
	return strconv.Atoi(os.Getenv(OtelAuditQueueSize))
}

func (p *prod) AROOperatorImage() string {
	return fmt.Sprintf("%s/aro:%s", p.acrDomain, version.GitCommit)
}

func (p *prod) populateVMSkus(ctx context.Context, resourceSkusClient compute.ResourceSkusClient) error {
	// Filtering is poorly documented, but currently (API version 2019-04-01)
	// it seems that the API returns all SKUs without a filter and with invalid
	// value in the filter.
	// Filtering gives significant optimisation: at the moment of writing,
	// we get ~1.2M response in eastus vs ~37M unfiltered (467 items vs 16618).
	filter := fmt.Sprintf("location eq '%s'", p.Location())
	skus, err := resourceSkusClient.List(ctx, filter)
	if err != nil {
		return err
	}

	p.vmskus = computeskus.FilterVMSizes(skus, p.Location())

	return nil
}

func (p *prod) ClusterGenevaLoggingAccount() string {
	return p.clusterGenevaLoggingAccount
}

func (p *prod) ClusterGenevaLoggingConfigVersion() string {
	return p.clusterGenevaLoggingConfigVersion
}

func (p *prod) ClusterGenevaLoggingEnvironment() string {
	return p.clusterGenevaLoggingEnvironment
}

func (p *prod) ClusterGenevaLoggingNamespace() string {
	return p.clusterGenevaLoggingNamespace
}

func (p *prod) ClusterGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate) {
	return p.clusterGenevaLoggingPrivateKey, p.clusterGenevaLoggingCertificate
}

func (p *prod) ClusterKeyvault() keyvault.Manager {
	return p.clusterKeyvault
}

func (p *prod) ClusterMsiKeyVaultName() string {
	return os.Getenv(KeyvaultPrefix) + ClusterMsiKeyVaultSuffix
}

func (p *prod) Domain() string {
	return os.Getenv("DOMAIN_NAME")
}

func (p *prod) FeatureIsSet(f Feature) bool {
	return p.features[f]
}

// TODO: Delete FPAuthorizer once the replace from track1 to track2 is done.
func (p *prod) FPAuthorizer(tenantID string, additionalTenants []string, scopes ...string) (autorest.Authorizer, error) {
	fpTokenCredential, err := p.FPNewClientCertificateCredential(tenantID, additionalTenants)
	if err != nil {
		return nil, err
	}

	return azidext.NewTokenCredentialAdapter(fpTokenCredential, scopes), nil
}

func (p *prod) FPClientID() string {
	return p.fpClientID
}

func (p *prod) Listen() (net.Listener, error) {
	return net.Listen("tcp", ":8443")
}

func (p *prod) GatewayDomains() []string {
	gatewayDomains := make([]string, len(p.gatewayDomains))

	copy(gatewayDomains, p.gatewayDomains)

	return gatewayDomains
}

func (p *prod) GatewayResourceGroup() string {
	return os.Getenv("GATEWAY_RESOURCEGROUP")
}

func (p *prod) ServiceKeyvault() keyvault.Manager {
	return p.serviceKeyvault
}

func (p *prod) VMSku(vmSize string) (*mgmtcompute.ResourceSku, error) {
	vmsku, found := p.vmskus[vmSize]
	if !found {
		return nil, fmt.Errorf("sku information not found for vm size %q", vmSize)
	}
	return vmsku, nil
}

func (p *prod) LiveConfig() liveconfig.Manager {
	return p.liveConfig
}

func (p *prod) FPNewClientCertificateCredential(tenantID string, additionalTenants []string) (*azidentity.ClientCertificateCredential, error) {
	fpPrivateKey, fpCertificates := p.fpCertificateRefresher.GetCertificates()

	options := p.Environment().ClientCertificateCredentialOptions(additionalTenants)
	credential, err := azidentity.NewClientCertificateCredential(tenantID, p.fpClientID, fpCertificates, fpPrivateKey, options)
	if err != nil {
		return nil, err
	}

	return credential, nil
}

func (p *prod) MsiRpEndpoint() string {
	return fmt.Sprintf("https://%s", os.Getenv("MSI_RP_ENDPOINT"))
}

func (p *prod) MsiDataplaneClientOptions() (*policy.ClientOptions, error) {
	armClientOptions := p.Environment().ArmClientOptions(ClientDebugLoggerMiddleware(p.log.WithField("client", "msi-dataplane")))
	clientOptions := armClientOptions.ClientOptions

	return &clientOptions, nil
}

func ClientDebugLoggerMiddleware(log *logrus.Entry) policy.Policy {
	return azureclient.PolicyFunc(func(req *policy.Request) (*http.Response, error) {
		log := log.WithFields(logrus.Fields{
			"method": req.Raw().Method,
			"url":    req.Raw().URL,
		})
		if req.Raw().Body != nil {
			body, err := io.ReadAll(req.Raw().Body)
			if err != nil {
				log.WithError(err).Error("error reading request body")
			}
			if err := req.Raw().Body.Close(); err != nil {
				log.WithError(err).Error("error closing request body")
			}
			log = log.WithField("body", string(body))
			req.Raw().Body = io.NopCloser(bytes.NewBuffer(body)) // reset body so the delegate can use it
		}
		log.Info("Sending request.")
		resp, err := req.Next()
		if err != nil {
			log.WithError(err).Error("Request errored.")
		} else if resp != nil {
			log = log.WithFields(logrus.Fields{
				"status": resp.StatusCode,
			})
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.WithError(err).Error("error reading response body")
			}
			if err := resp.Body.Close(); err != nil {
				log.WithError(err).Error("error closing response body")
			}
			// n.b.: we only send one request now, this is best-effort but would need to be updated if we use other methods
			var responseBody string
			if resp.StatusCode == http.StatusOK {
				response := dataplane.ManagedIdentityCredentials{}
				if err := json.Unmarshal(body, &response); err != nil {
					log.WithError(err).Error("error unmarshalling response body")
				} else {
					censorCredentials(&response)
					censored, err := json.Marshal(response)
					if err != nil {
						log.WithError(err).Error("error marshalling response body after censoring")
					}
					responseBody = string(censored)
				}
			} else {
				// error codes don't have anything in them that we need to censor
				responseBody = string(body)
			}
			log = log.WithField("body", responseBody)
			resp.Body = io.NopCloser(bytes.NewBuffer(body)) // reset body so the upstream round-trippers can use it
		}
		log.Info("Received response.")

		return resp, err
	})
}

func censorCredentials(input *dataplane.ManagedIdentityCredentials) {
	input.ClientSecret = nil
	for i := 0; i < len(input.DelegatedResources); i++ {
		if input.DelegatedResources[i].ImplicitIdentity != nil {
			input.DelegatedResources[i].ImplicitIdentity.ClientSecret = nil
		}
		for j := 0; j < len(input.DelegatedResources[i].ExplicitIdentities); j++ {
			input.DelegatedResources[i].ExplicitIdentities[j].ClientSecret = nil
		}
	}
	if input.ExplicitIdentities != nil {
		for i := 0; i < len(input.ExplicitIdentities); i++ {
			input.ExplicitIdentities[i].ClientSecret = nil
		}
	}
}

func (p *prod) MockMSIResponses(msiResourceId *arm.ResourceID) dataplane.ClientFactory {
	return &mockFactory{aadHost: p.Environment().Cloud.ActiveDirectoryAuthorityHost, msiResourceId: msiResourceId.String()}
}

func MockMSIResponses(aadHost string, msiResourceId *arm.ResourceID) dataplane.ClientFactory {
	return &mockFactory{aadHost: aadHost, msiResourceId: msiResourceId.String()}
}

type mockFactory struct {
	aadHost       string
	msiResourceId string
}

var _ dataplane.ClientFactory = (*mockFactory)(nil)

func (m *mockFactory) NewClient(identityURL string) (dataplane.Client, error) {
	return &mockClient{
		aadHost:       m.aadHost,
		msiResourceId: m.msiResourceId,
	}, nil
}

type mockClient struct {
	aadHost       string
	msiResourceId string
}

var _ dataplane.Client = (*mockClient)(nil)

func (m *mockClient) DeleteSystemAssignedIdentity(ctx context.Context) error {
	panic("not yet implemented")
}

func (m *mockClient) GetSystemAssignedIdentityCredentials(ctx context.Context) (*dataplane.ManagedIdentityCredentials, error) {
	panic("not yet implemented")
}

func (m *mockClient) GetUserAssignedIdentitiesCredentials(ctx context.Context, request dataplane.UserAssignedIdentitiesRequest) (*dataplane.ManagedIdentityCredentials, error) {
	keysToValidate := []string{
		"MOCK_MSI_CLIENT_ID",
		"MOCK_MSI_OBJECT_ID",
		"MOCK_MSI_CERT",
		"MOCK_MSI_TENANT_ID",
	}

	if err := ValidateVars(keysToValidate...); err != nil {
		return nil, err
	}

	placeholder := "placeholder"
	return &dataplane.ManagedIdentityCredentials{
		ExplicitIdentities: []dataplane.UserAssignedIdentityCredentials{
			{
				ClientID:                   pointerutils.ToPtr(os.Getenv("MOCK_MSI_CLIENT_ID")),
				ClientSecret:               pointerutils.ToPtr(os.Getenv("MOCK_MSI_CERT")),
				TenantID:                   pointerutils.ToPtr(os.Getenv("MOCK_MSI_TENANT_ID")),
				ObjectID:                   pointerutils.ToPtr(os.Getenv("MOCK_MSI_OBJECT_ID")),
				ResourceID:                 pointerutils.ToPtr(m.msiResourceId),
				AuthenticationEndpoint:     pointerutils.ToPtr(m.aadHost),
				CannotRenewAfter:           &placeholder,
				ClientSecretURL:            &placeholder,
				MtlsAuthenticationEndpoint: &placeholder,
				NotAfter:                   &placeholder,
				NotBefore:                  &placeholder,
				RenewAfter:                 &placeholder,
				CustomClaims: &dataplane.CustomClaims{
					XMSAzNwperimid: []string{placeholder},
					XMSAzTm:        &placeholder,
				},
			},
		},
	}, nil
}

func (m *mockClient) MoveIdentity(ctx context.Context, request dataplane.MoveIdentityRequest) (*dataplane.MoveIdentityResponse, error) {
	panic("not yet implemented")
}
