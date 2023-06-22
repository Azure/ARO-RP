package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

// Core collects basic configuration information which is expected to be
// available on any PROD service VMSS (i.e. instance metadata, MSI authorizer,
// etc.)
type Core interface {
	IsLocalDevelopmentMode() bool
	IsCI() bool
	NewMSIAuthorizer(MSIContext, ...string) (autorest.Authorizer, error)
	NewLiveConfigManager(context.Context) (liveconfig.Manager, error)
	instancemetadata.InstanceMetadata
}

type core struct {
	instancemetadata.InstanceMetadata

	isLocalDevelopmentMode bool
	isCI                   bool
}

func (c *core) IsLocalDevelopmentMode() bool {
	return c.isLocalDevelopmentMode
}

func (c *core) IsCI() bool {
	return c.isCI
}

func (c *core) NewLiveConfigManager(ctx context.Context) (liveconfig.Manager, error) {
	msiAuthorizer, err := c.NewMSIAuthorizer(MSIContextRP, c.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	mcc := containerservice.NewManagedClustersClient(c.Environment(), c.SubscriptionID(), msiAuthorizer)

	if c.isLocalDevelopmentMode {
		return liveconfig.NewDev(c.Location(), mcc), nil
	}

	return liveconfig.NewProd(c.Location(), mcc), nil
}

func NewCore(ctx context.Context, log *logrus.Entry) (Core, error) {
	// assign results of package-level functions to struct's environment flags
	isLocalDevelopmentMode := IsLocalDevelopmentMode()
	isCI := IsCI()
	if isLocalDevelopmentMode {
		log.Info("running in local development mode")
	}

	im, err := instancemetadata.New(ctx, log, isLocalDevelopmentMode)
	if err != nil {
		return nil, err
	}

	log.Infof("InstanceMetadata: running on %s", im.Environment().Name)

	return &core{
		InstanceMetadata: im,

		isLocalDevelopmentMode: isLocalDevelopmentMode,
		isCI:                   isCI,
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

	return &core{
		InstanceMetadata:       im,
		isLocalDevelopmentMode: isLocalDevelopmentMode,
	}, nil
}
