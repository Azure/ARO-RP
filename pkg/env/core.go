package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/deploy/generator"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/keyvault"
	"github.com/Azure/ARO-RP/pkg/util/rpauthorizer"
)

type Core interface {
	DeploymentMode() deployment.Mode
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer

	GetBase64Secret(context.Context, string) ([]byte, error)
	GetCertificateSecret(context.Context, string) (*rsa.PrivateKey, []*x509.Certificate, error)
}

type core struct {
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer

	deploymentMode  deployment.Mode
	servicekeyvault keyvault.Manager
}

func (c *core) DeploymentMode() deployment.Mode {
	return c.deploymentMode
}

func (c *core) GetBase64Secret(ctx context.Context, secretName string) ([]byte, error) {
	return c.servicekeyvault.GetBase64Secret(ctx, secretName)
}

func (c *core) GetCertificateSecret(ctx context.Context, secretName string) (*rsa.PrivateKey, []*x509.Certificate, error) {
	return c.servicekeyvault.GetCertificateSecret(ctx, secretName)
}

func NewCore(ctx context.Context, log *logrus.Entry) (Core, error) {
	deploymentMode := deployment.NewMode()
	log.Infof("running in %s mode", deploymentMode)

	instancemetadata, err := instancemetadata.New(ctx, deploymentMode)
	if err != nil {
		return nil, err
	}

	rpauthorizer, err := rpauthorizer.New(deploymentMode)
	if err != nil {
		return nil, err
	}

	rpKVAuthorizer, err := rpauthorizer.NewRPAuthorizer(instancemetadata.Environment().ResourceIdentifiers.KeyVault)
	if err != nil {
		return nil, err
	}

	serviceKeyvaultURI, err := keyvault.Find(ctx, instancemetadata, rpauthorizer, generator.ServiceKeyVaultTagValue)
	if err != nil {
		return nil, err
	}

	return &core{
		InstanceMetadata: instancemetadata,
		RPAuthorizer:     rpauthorizer,

		deploymentMode:  deploymentMode,
		servicekeyvault: keyvault.NewManager(rpKVAuthorizer, serviceKeyvaultURI),
	}, nil
}
