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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/msi-dataplane/pkg/dataplane"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcertificates"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
	"github.com/Azure/ARO-RP/pkg/util/miseadapter"
)

type Feature int

// At least to start with, features are intended to be used so that the
// production default is not set (in production RP_FEATURES is unset).
const (
	FeatureDisableDenyAssignments Feature = iota
	FeatureDisableSignedCertificates
	FeatureEnableDevelopmentAuthorizer
	FeatureRequireD2sWorkers
	FeatureDisableReadinessDelay
	FeatureRequireOIDCStorageWebEndpoint
	FeatureUseMockMsiRp
	FeatureEnableMISE
	FeatureEnforceMISE
)

const (
	RPDevARMSecretName               = "dev-arm"
	RPFirstPartySecretName           = "rp-firstparty"
	RPServerSecretName               = "rp-server"
	ClusterLoggingSecretName         = "cluster-mdsd"
	EncryptionSecretName             = "encryption-key"
	EncryptionSecretV2Name           = "encryption-key-v2"
	FrontendEncryptionSecretName     = "fe-encryption-key"
	FrontendEncryptionSecretV2Name   = "fe-encryption-key-v2"
	PortalServerSecretName           = "portal-server"
	PortalServerClientSecretName     = "portal-client"
	PortalServerSessionKeySecretName = "portal-session-key"
	PortalServerSSHKeySecretName     = "portal-sshkey"
	ClusterKeyvaultSuffix            = "-cls"
	GatewayKeyvaultSuffix            = "-gwy"
	PortalKeyvaultSuffix             = "-por"
	ServiceKeyvaultSuffix            = "-svc"
	ClusterMsiKeyVaultSuffix         = "-msi"
	RPPrivateEndpointPrefix          = "rp-pe-"
	ProxyHostName                    = "PROXY_HOSTNAME"
)

// Interface is clunky and somewhat legacy and only used in the RP codebase (not
// monitor/portal/gateway, etc.).  It is a grab-bag of items which modify RP
// behaviour depending on where it is running (dev, prod, etc.)  Outside of the
// RP codebase, use Core.  Ideally we might break Interface into smaller pieces,
// either closer to their point of use, or maybe using dependency injection. Try
// to remove methods, not add more.  A refactored approach to configuration is
// generally necessary across all of the ARO services; dealing with Interface
// should be part of that.
type Interface interface {
	Core
	proxy.Dialer
	ARMHelper

	InitializeAuthorizers() error
	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
	MISEAuthorizer() miseadapter.MISEAdapter
	ClusterGenevaLoggingAccount() string
	ClusterGenevaLoggingConfigVersion() string
	ClusterGenevaLoggingEnvironment() string
	ClusterGenevaLoggingNamespace() string
	ClusterGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClusterKeyvault() azsecrets.Client
	ClusterMsiKeyVaultName() string
	Domain() string
	FeatureIsSet(Feature) bool
	// TODO: Delete FPAuthorizer once the replace from track1 to track2 is done.
	FPAuthorizer(string, []string, ...string) (autorest.Authorizer, error)
	FPNewClientCertificateCredential(string, []string) (*azidentity.ClientCertificateCredential, error)
	FPClientID() string
	Listen() (net.Listener, error)
	GatewayDomains() []string
	GatewayResourceGroup() string
	ServiceKeyvault() azsecrets.Client
	ACRResourceID() string
	ACRDomain() string
	OIDCStorageAccountName() string
	OIDCEndpoint() string
	OIDCKeyBitSize() int
	OtelAuditQueueSize() (int, error)
	MsiRpEndpoint() string
	MsiDataplaneClientOptions() (*policy.ClientOptions, error)
	MockMSIResponses(msiResourceId *arm.ResourceID) dataplane.ClientFactory
	AROOperatorImage() string
	LiveConfig() liveconfig.Manager
	ClusterCertificates() azcertificates.Client
}

func NewEnv(ctx context.Context, log *logrus.Entry, component ServiceComponent) (Interface, error) {
	if IsLocalDevelopmentMode() {
		if err := ValidateVars(ProxyHostName); err != nil {
			return nil, err
		}
		return newDev(ctx, log, component)
	}

	return newProd(ctx, log, component)
}

func IsLocalDevelopmentMode() bool {
	return strings.EqualFold(os.Getenv("RP_MODE"), "development")
}

func IsCI() bool {
	return strings.EqualFold(os.Getenv("CI"), "true")
}

// ValidateVars iterates over all the elements of vars and
// if it does not exist an environment variable with that name, it will return an error.
// Otherwise it returns nil.
func ValidateVars(vars ...string) error {
	var err error

	for _, envName := range vars {
		if envValue, found := os.LookupEnv(envName); !found || envValue == "" {
			err = multierror.Append(fmt.Errorf("environment variable %q unset", envName), err)
		}
	}
	return err
}
