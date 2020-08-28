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

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
)

type environmentType uint8

const (
	environmentTypeProduction environmentType = iota
	environmentTypeDevelopment
	environmentTypeIntegration
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
	Lite
	ServiceKeyvaultInterface

	InitializeAuthorizers() error
	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
	CreateARMResourceGroupRoleAssignment(context.Context, refreshable.Authorizer, string) error
	ClustersGenevaLoggingConfigVersion() string
	ClustersGenevaLoggingEnvironment() string
	ClustersGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClustersKeyvaultURI() string
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
	ShouldDeployDenyAssignment() bool
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if isDevelopment() {
		log.Warn("running in development mode")
	}

	im, err := newInstanceMetadata(ctx)
	if err != nil {
		return nil, err
	}

	if isDevelopment() {
		return newDev(ctx, log, im)
	}

	if strings.ToLower(os.Getenv("RP_MODE")) == "int" {
		log.Warn("running in int mode")
		return newInt(ctx, log, im)
	}

	return newProd(ctx, log, im)
}

func newInstanceMetadata(ctx context.Context) (instancemetadata.InstanceMetadata, error) {
	if isDevelopment() {
		return instancemetadata.NewDev(), nil
	}

	return instancemetadata.NewProd(ctx)
}

func isDevelopment() bool {
	return strings.ToLower(os.Getenv("RP_MODE")) == "development"
}
