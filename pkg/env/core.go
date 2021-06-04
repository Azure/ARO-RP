package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
)

// Core collects basic configuration information which is expected to be
// available on any PROD service VMSS (i.e. instance metadata, MSI authorizer,
// etc.)
type Core interface {
	IsLocalDevelopmentMode() bool
	NewMSIAuthorizer(MSIContext, string) (autorest.Authorizer, error)
	instancemetadata.InstanceMetadata
}

type core struct {
	instancemetadata.InstanceMetadata

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

	im, err := instancemetadata.New(ctx, log, isLocalDevelopmentMode)
	if err != nil {
		return nil, err
	}

	err = validateCloudEnvironment(im.Environment().Name)
	if err != nil {
		return nil, err
	}
	log.Infof("InstanceMetadata: environment=%s subscriptionID=%s tenantID=%s location=%s resourceGroup=%s hostname=%s", im.Environment().Name, im.SubscriptionID(), im.TenantID(), im.Location(), im.ResourceGroup(), im.Hostname())

	return &core{
		InstanceMetadata: im,

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
