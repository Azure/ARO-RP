package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

const (
	RPFirstPartySecretName           = "rp-firstparty"
	RPServerSecretName               = "rp-server"
	ClusterLoggingSecretName         = "cluster-mdsd"
	EncryptionSecretName             = "encryption-key"
	FrontendEncryptionSecretName     = "fe-encryption-key"
	RPLoggingSecretName              = "rp-mdsd"
	RPMonitoringSecretName           = "rp-mdm"
	PortalServerSecretName           = "portal-server"
	PortalServerClientSecretName     = "portal-client"
	PortalServerSessionKeySecretName = "portal-session-key"
	PortalServerSSHKeySecretName     = "portal-sshkey"
	ClusterKeyvaultSuffix            = "-cls"
	PortalKeyvaultSuffix             = "-por"
	ServiceKeyvaultSuffix            = "-svc"
	RPPrivateEndpointPrefix          = "rp-pe-"
)

type Interface interface {
	Core
	proxy.Dialer

	InitializeAuthorizers() error
	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
	EnsureARMResourceGroupRoleAssignment(context.Context, refreshable.Authorizer, string) error
	ClusterGenevaLoggingConfigVersion() string
	ClusterGenevaLoggingEnvironment() string
	ClusterGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClusterKeyvault() keyvault.Manager
	Domain() string
	FPAuthorizer(string, string) (refreshable.Authorizer, error)
	Listen() (net.Listener, error)
	ServiceKeyvault() keyvault.Manager
	Zones(vmSize string) ([]string, error)
	ACRResourceID() string
	ACRDomain() string
	AROOperatorImage() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	switch deployment.NewMode() {
	case deployment.Development:
		return newDev(ctx, log)
	case deployment.Integration:
		return newInt(ctx, log)
	default:
		return newProd(ctx, log)
	}
}
