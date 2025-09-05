package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/fips140"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"

	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/pkg/util/liveconfig"
)

// ServiceName is the name of the runtime service (e.g. gateway, monitor)
type ServiceName string

const (
	SERVICE_RP                  ServiceName = "RP"
	SERVICE_GATEWAY             ServiceName = "GATEWAY"
	SERVICE_MONITOR             ServiceName = "MONITOR"
	SERVICE_OPERATOR            ServiceName = "OPERATOR"
	SERVICE_MIRROR              ServiceName = "MIRROR"
	SERVICE_PORTAL              ServiceName = "PORTAL"
	SERVICE_UPDATE_OCP_VERSIONS ServiceName = "UPDATE_OCP_VERSIONS"
	SERVICE_UPDATE_ROLE_SETS    ServiceName = "UPDATE_ROLE_SETS"
	SERVICE_DEPLOY              ServiceName = "DEPLOY"
	SERVICE_TOOLING             ServiceName = "TOOLING"
	SERVICE_MIMO_SCHEDULER      ServiceName = "MIMO_SCHEDULER"
	SERVICE_MIMO_ACTUATOR       ServiceName = "MIMO_ACTUATOR"
	SERVICE_E2E                 ServiceName = "E2E"
)

// Core collects basic configuration information which is expected to be
// available on any PROD service VMSS (i.e. instance metadata, MSI authorizer,
// etc.)
type Core interface {
	IsLocalDevelopmentMode() bool
	IsCI() bool
	NewMSITokenCredential() (azcore.TokenCredential, error)
	NewMSIAuthorizer(scope string) (autorest.Authorizer, error)
	NewLiveConfigManager(context.Context) (liveconfig.Manager, error)
	instancemetadata.InstanceMetadata

	Service() string
	Logger() *logrus.Entry
	LoggerForComponent(string) *logrus.Entry
}

type core struct {
	instancemetadata.InstanceMetadata

	isLocalDevelopmentMode bool
	isCI                   bool

	service    ServiceName
	serviceLog *logrus.Entry

	msiAuthorizers map[string]autorest.Authorizer
}

func (c *core) IsLocalDevelopmentMode() bool {
	return c.isLocalDevelopmentMode
}

func (c *core) IsCI() bool {
	return c.isCI
}

func (c *core) Service() string {
	return strings.ToLower(string(c.service))
}

func (c *core) Logger() *logrus.Entry {
	return c.serviceLog
}

// LoggerForComponent creates a logger with the "component" field set. This
// should be used when passing a logger to an internal component such as the
// database or serving components such as the frontend.
func (c *core) LoggerForComponent(component string) *logrus.Entry {
	return c.serviceLog.WithField("component", component)
}

func (c *core) NewLiveConfigManager(ctx context.Context) (liveconfig.Manager, error) {
	credential, err := c.NewMSITokenCredential()
	if err != nil {
		return nil, err
	}

	mcc, err := utilcontainerservice.NewDefaultManagedClustersClient(c.Environment(), c.SubscriptionID(), credential)
	if err != nil {
		return nil, err
	}

	if c.isLocalDevelopmentMode {
		return liveconfig.NewDev(c.Location(), mcc), nil
	}

	return liveconfig.NewProd(c.Location(), mcc), nil
}

func NewCore(ctx context.Context, _log *logrus.Entry, service ServiceName) (Core, error) {
	// set the service field on the logger (e.g. monitor, gateway)
	log := _log.WithField("service", strings.ReplaceAll(strings.ToLower(string(service)), "_", "-"))

	// assign results of package-level functions to struct's environment flags
	isLocalDevelopmentMode := IsLocalDevelopmentMode()
	isCI := IsCI()
	if isLocalDevelopmentMode {
		log.Info("running in local development mode")
	}

	// https://go.dev/doc/security/fips140
	if fips140.Enabled() {
		log.Infof("running in FIPS 140-3 mode")
	} else {
		log.Infof("running without FIPS 140-3 mode")
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
		service:                service,
		serviceLog:             log,
		msiAuthorizers:         map[string]autorest.Authorizer{},
	}, nil
}

// NewCoreForCI returns an env.Core which respects RP_MODE but always uses
// AZURE_* environment variables instead of IMDS.  This is used for entrypoints
// which may run on CI VMs.  CI VMs don't currently have MSI and hence cannot
// resolve their tenant ID, and also may access resources in a different tenant
// (e.g. AME).
func NewCoreForCI(ctx context.Context, _log *logrus.Entry, service ServiceName) (Core, error) {
	// set the service field on the logger (e.g. monitor, gateway)
	log := _log.WithField("service", strings.ReplaceAll(strings.ToLower(string(service)), "_", "-"))
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
		msiAuthorizers:         map[string]autorest.Authorizer{},
		service:                service,
		serviceLog:             log,
	}, nil
}
