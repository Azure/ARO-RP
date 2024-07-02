package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

type ServiceComponent string

const (
	COMPONENT_RP                  ServiceComponent = "RP"
	COMPONENT_GATEWAY             ServiceComponent = "GATEWAY"
	COMPONENT_MONITOR             ServiceComponent = "MONITOR"
	COMPONENT_OPERATOR            ServiceComponent = "OPERATOR"
	COMPONENT_MIRROR              ServiceComponent = "MIRROR"
	COMPONENT_PORTAL              ServiceComponent = "PORTAL"
	COMPONENT_UPDATE_OCP_VERSIONS ServiceComponent = "UPDATE_OCP_VERSIONS"
	COMPONENT_UPDATE_ROLE_SETS    ServiceComponent = "UPDATE_ROLE_SETS"
	COMPONENT_DEPLOY              ServiceComponent = "DEPLOY"
	COMPONENT_TOOLING             ServiceComponent = "TOOLING"
)

// Core collects basic configuration information which is expected to be
// available on any PROD service VMSS (i.e. instance metadata, MSI authorizer,
// etc.)
type Core interface {
	IsLocalDevelopmentMode() bool
	IsCI() bool
	NewMSITokenCredential() (azcore.TokenCredential, error)
	NewMSIAuthorizer(...string) (autorest.Authorizer, error)
	NewLiveConfigManager(context.Context) (liveconfig.Manager, error)
	instancemetadata.InstanceMetadata

	Component() string
	Logger() *logrus.Entry
}

type core struct {
	instancemetadata.InstanceMetadata

	isLocalDevelopmentMode bool
	isCI                   bool

	component    ServiceComponent
	componentLog *logrus.Entry
}

func (c *core) IsLocalDevelopmentMode() bool {
	return c.isLocalDevelopmentMode
}

func (c *core) IsCI() bool {
	return c.isCI
}

func (c *core) Component() string {
	return string(c.component)
}

func (c *core) Logger() *logrus.Entry {
	return c.componentLog
}

func (c *core) NewLiveConfigManager(ctx context.Context) (liveconfig.Manager, error) {
	msiAuthorizer, err := c.NewMSIAuthorizer(c.Environment().ResourceManagerScope)
	if err != nil {
		return nil, err
	}

	mcc := containerservice.NewManagedClustersClient(c.Environment(), c.SubscriptionID(), msiAuthorizer)

	if c.isLocalDevelopmentMode {
		return liveconfig.NewDev(c.Location(), mcc), nil
	}

	return liveconfig.NewProd(c.Location(), mcc), nil
}

func NewCore(ctx context.Context, log *logrus.Entry, component ServiceComponent) (Core, error) {
	// assign results of package-level functions to struct's environment flags
	isLocalDevelopmentMode := IsLocalDevelopmentMode()
	isCI := IsCI()
	componentLog := log.WithField("component", strings.ReplaceAll(strings.ToLower(string(component)), "_", "-"))
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
		component:              component,
		componentLog:           componentLog,
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
