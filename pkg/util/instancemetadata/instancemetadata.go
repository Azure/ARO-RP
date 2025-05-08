package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

type InstanceMetadata interface {
	Hostname() string
	TenantID() string
	SubscriptionID() string
	Location() string
	ResourceGroup() string
	Environment() *azureclient.AROEnvironment
}

type instanceMetadata struct {
	hostname       string
	tenantID       string
	subscriptionID string
	location       string
	resourceGroup  string
	environment    *azureclient.AROEnvironment
}

func (im *instanceMetadata) Hostname() string {
	return im.hostname
}

func (im *instanceMetadata) TenantID() string {
	return im.tenantID
}

func (im *instanceMetadata) SubscriptionID() string {
	return im.subscriptionID
}

func (im *instanceMetadata) Location() string {
	return im.location
}

func (im *instanceMetadata) ResourceGroup() string {
	return im.resourceGroup
}

func (im *instanceMetadata) Environment() *azureclient.AROEnvironment {
	return im.environment
}

// New returns a new InstanceMetadata for the given mode, environment, and deployment system
func New(ctx context.Context, log *logrus.Entry, isLocalDevelopmentMode bool) (InstanceMetadata, error) {
	if isLocalDevelopmentMode {
		log.Info("creating development InstanceMetadata")
		return NewDev(true)
	}

	if os.Getenv("AZURE_EV2") != "" {
		log.Info("creating InstanceMetadata from Environment")
		return newProdFromEnv(ctx)
	}

	log.Info("creating InstanceMetadata from Azure Instance Metadata Service (AIMS)")
	return newProd(ctx)
}
