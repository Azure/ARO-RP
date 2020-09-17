package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

const (
	RPFirstPartySecretName       = "rp-firstparty"
	RPServerSecretName           = "rp-server"
	ClusterLoggingSecretName     = "cluster-mdsd"
	EncryptionSecretName         = "encryption-key"
	FrontendEncryptionSecretName = "fe-encryption-key"
	RPLoggingSecretName          = "rp-mdsd"
	RPMonitoringSecretName       = "rp-mdm"
)

type Interface interface {
	Core

	InitializeAuthorizers() error
	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
	CreateARMResourceGroupRoleAssignment(context.Context, refreshable.Authorizer, string) error
	ClustersGenevaLoggingConfigVersion() string
	ClustersGenevaLoggingEnvironment() string
	ClustersGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClustersKeyvaultURI() string
	CosmosDB() (string, string)
	DatabaseName() string
	DialContext(context.Context, string, string) (net.Conn, error)
	Domain() string
	FPAuthorizer(string, string) (refreshable.Authorizer, error)
	Listen() (net.Listener, error)
	ManagedDomain(string) (string, error)
	MetricsSocketPath() string
	Zones(vmSize string) ([]string, error)
	ACRResourceID() string
	ACRName() string
	AROOperatorImage() string
	E2EStorageAccountName() string
	E2EStorageAccountRGName() string
	E2EStorageAccountSubID() string
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
