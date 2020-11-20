package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/rpauthorizer"
)

type Core interface {
	DeploymentMode() deployment.Mode
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer
}

type core struct {
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer

	deploymentMode deployment.Mode
}

func (c *core) DeploymentMode() deployment.Mode {
	return c.deploymentMode
}

func NewCore(ctx context.Context, log *logrus.Entry) (Core, error) {
	deploymentMode := deployment.NewMode()
	log.Infof("running in %s mode", deploymentMode)

	im, err := instancemetadata.New(ctx, deploymentMode)
	if err != nil {
		return nil, err
	}

	switch im.Environment().Name {
	case azure.PublicCloud.Name, azure.USGovernmentCloud.Name:
	default:
		return nil, errors.New("unsupported Azure cloud environment")
	}

	rpauthorizer, err := rpauthorizer.New(deploymentMode, im)
	if err != nil {
		return nil, err
	}

	return &core{
		InstanceMetadata: im,
		RPAuthorizer:     rpauthorizer,

		deploymentMode: deploymentMode,
	}, nil
}
