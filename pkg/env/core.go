package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

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

	instancemetadata, err := instancemetadata.New(ctx, deploymentMode)
	if err != nil {
		return nil, err
	}

	rpauthorizer, err := rpauthorizer.New(deploymentMode)
	if err != nil {
		return nil, err
	}

	return &core{
		InstanceMetadata: instancemetadata,
		RPAuthorizer:     rpauthorizer,

		deploymentMode: deploymentMode,
	}, nil
}
