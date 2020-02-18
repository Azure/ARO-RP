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
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

type Interface interface {
	instancemetadata.InstanceMetadata

	ArmClientAuthorizer() clientauthorizer.ClientAuthorizer
	AdminClientAuthorizer() clientauthorizer.ClientAuthorizer
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
	GenevaLoggingSecret() (*rsa.PrivateKey, []*x509.Certificate, error)
	GenevaLoggingEnvironment() string
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return newDev(ctx, log, instancemetadata.NewDev(), clientauthorizer.NewAll(), clientauthorizer.NewAll())
	}

	im, err := instancemetadata.NewProd()
	if err != nil {
		return nil, err
	}

	if strings.ToLower(os.Getenv("RP_MODE")) == "int" {
		log.Warn("running in int mode")
		return newInt(ctx, log, im, clientauthorizer.NewARM(log), clientauthorizer.NewAdmin(log))
	}

	for _, key := range []string{
		"MDM_ACCOUNT",
		"MDM_NAMESPACE",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	return newProd(ctx, log, im, clientauthorizer.NewARM(log), clientauthorizer.NewAdmin(log))
}
