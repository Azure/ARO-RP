package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/rpauthorizer"
)

type Core interface {
	IsLocalDevelopmentMode() bool
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer
}

type core struct {
	instancemetadata.InstanceMetadata
	rpauthorizer.RPAuthorizer

	isLocalDevelopmentMode bool
}

func (c *core) IsLocalDevelopmentMode() bool {
	return c.isLocalDevelopmentMode
}

func NewCore(ctx context.Context, log *logrus.Entry) (Core, error) {
	isLocalDevelopmentMode := IsLocalDevelopmentMode()
	if isLocalDevelopmentMode {
		log.Info("running in local development mode")
	}

	im, err := instancemetadata.New(ctx, isLocalDevelopmentMode)
	if err != nil {
		return nil, err
	}

	err = validateCloudEnvironment(im.Environment().Name)
	if err != nil {
		return nil, err
	}
	log.Infof("running on %s", im.Environment().Name)

	rpauthorizer, err := rpauthorizer.New(isLocalDevelopmentMode, im)
	if err != nil {
		return nil, err
	}

	return &core{
		InstanceMetadata: im,
		RPAuthorizer:     rpauthorizer,

		isLocalDevelopmentMode: isLocalDevelopmentMode,
	}, nil
}

// NewCoreForCI returns an env.Core which respects RP_MODE but always uses
// AZURE_* environment variables instead of IMDS.  This is used for entrypoints
// which may run on CI VMs.  CI VMs don't currently have MSI and hence cannot
// resolve their tenant ID, and also may access resources in a different tenant
// (e.g. AME).
func NewCoreForCI(ctx context.Context, log *logrus.Entry) (Core, error) {
	isLocalDevelopmentMode := IsLocalDevelopmentMode()
	if isLocalDevelopmentMode {
		log.Info("running in local development mode")
	}

	im, err := instancemetadata.NewDev(false)
	if err != nil {
		return nil, err
	}

	err = validateCloudEnvironment(im.Environment().Name)
	if err != nil {
		return nil, err
	}

	return &core{
		InstanceMetadata:       im,
		isLocalDevelopmentMode: isLocalDevelopmentMode,
	}, nil
}

func validateCloudEnvironment(name string) error {
	switch name {
	case azure.PublicCloud.Name, azure.USGovernmentCloud.Name:
		return nil
	default:
		return errors.New("unsupported Azure cloud environment")
	}
}
