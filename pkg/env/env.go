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

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

const (
	RPFirstPartySecretName   = "rp-firstparty"
	RPServerSecretName       = "rp-server"
	ClusterLoggingSecretName = "cluster-mdsd"
	EncryptionSecretName     = "encryption-key"
	RPLoggingSecretName      = "rp-mdsd"
	RPMonitoringSecretName   = "rp-mdm"
)

type Interface interface {
	instancemetadata.InstanceMetadata

	InitializeAuthorizers() error
	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
	ClustersGenevaLoggingConfigVersion() string
	ClustersGenevaLoggingEnvironment() string
	ClustersGenevaLoggingSecret() (*rsa.PrivateKey, *x509.Certificate)
	ClustersKeyvaultURI() string
	CosmosDB() (string, string)
	DatabaseName() string
	DialContext(context.Context, string, string) (net.Conn, error)
	Domain() string
	FPAuthorizer(string, string) (autorest.Authorizer, error)
	GetCertificateSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
	GetSecret(context.Context, string) ([]byte, error)
	Listen() (net.Listener, error)
	ManagedDomain(string) (string, error)
	MetricsSocketPath() string
	Zones(vmSize string) ([]string, error)
	ACRResourceID() string
	ACRName() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return newDev(ctx, log, instancemetadata.NewDev())
	}

	im, err := instancemetadata.NewProd()
	if err != nil {
		return nil, err
	}

	if strings.ToLower(os.Getenv("RP_MODE")) == "int" {
		log.Warn("running in int mode")
		return newInt(ctx, log, im)
	}

	return newProd(ctx, log, im)
}
