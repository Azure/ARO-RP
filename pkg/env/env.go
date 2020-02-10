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

type Interface interface {
	clientauthorizer.ClientAuthorizer
	instancemetadata.InstanceMetadata

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
	VnetName() string
	Zones(vmSize string) ([]string, error)
}

func NewEnv(ctx context.Context, log *logrus.Entry) (Interface, error) {
	if strings.ToLower(os.Getenv("RP_MODE")) == "development" {
		log.Warn("running in development mode")
		return newDev(ctx, log, instancemetadata.NewDev(), clientauthorizer.NewAll())
	}

	im, err := instancemetadata.NewProd()
	if err != nil {
		return nil, err
	}

	return newProd(ctx, log, im, clientauthorizer.NewARM(log))
}
