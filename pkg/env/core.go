package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
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

	ServiceKeyvault() keyvault.Manager
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

func (c *core) ServiceKeyvault() keyvault.Manager {
	return c.servicekeyvault
}

func NewCore(ctx context.Context, log *logrus.Entry) (Core, error) {
	deploymentMode := deployment.NewMode()

	switch deploymentMode {
	case deployment.Development:
		log.Warn("running in development mode")
	case deployment.Integration:
		log.Warn("running in int mode")
	}

	instancemetadata, err := instancemetadata.New(ctx, deploymentMode)
	if err != nil {
		return nil, err
	}

	rpauthorizer, err := rpauthorizer.New(deploymentMode)
	if err != nil {
		return nil, err
	}

	rpKVAuthorizer, err := rpauthorizer.NewRPAuthorizer(azure.PublicCloud.ResourceIdentifiers.KeyVault)
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
