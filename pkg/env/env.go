package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type Feature int

// At least to start with, features are intended to be used so that the
// production default is not set (in production RP_FEATURES is unset).
const (
	FeatureDisableDenyAssignments Feature = iota
	FeatureDisableSignedCertificates
	FeatureEnableDevelopmentAuthorizer
	FeatureRequireD2sV3Workers
	FeatureDisableReadinessDelay
)

const (
	RPDevARMSecretName               = "dev-arm"
	RPFirstPartySecretName           = "rp-firstparty"
	RPServerSecretName               = "rp-server"
	ClusterLoggingSecretName         = "cluster-mdsd"
	EncryptionSecretName             = "encryption-key"
	FrontendEncryptionSecretName     = "fe-encryption-key"
	DBTokenServerSecretName          = "dbtoken-server"
	RPLoggingSecretName              = "rp-mdsd"
	RPMonitoringSecretName           = "rp-mdm"
	PortalServerSecretName           = "portal-server"
	PortalServerClientSecretName     = "portal-client"
	PortalServerSessionKeySecretName = "portal-session-key"
	PortalServerSSHKeySecretName     = "portal-sshkey"
	ClusterKeyvaultSuffix            = "-cls"
	DBTokenKeyvaultSuffix            = "-dbt"
	PortalKeyvaultSuffix             = "-por"
	ServiceKeyvaultSuffix            = "-svc"
	RPPrivateEndpointPrefix          = "rp-pe-"
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
	ClusterGenevaLoggingConfigVersion() string
	ClusterGenevaLoggingEnvironment() string
	ClusterGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClusterKeyvault() keyvault.Manager
	Domain() string
	FeatureIsSet(Feature) bool
	FPAuthorizer(string, string) (refreshable.Authorizer, error)
	FPClientID() string
	Listen() (net.Listener, error)
	ServiceKeyvault() keyvault.Manager
	Zones(vmSize string) ([]string, error)
	ACRResourceID() string
	ACRDomain() string
	AROOperatorImage() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if IsLocalDevelopmentMode() {
		return newDev(ctx, log)
	}

	return newProd(ctx, log)
}

func IsLocalDevelopmentMode() bool {
	return strings.EqualFold(os.Getenv("RP_MODE"), "development")
}
