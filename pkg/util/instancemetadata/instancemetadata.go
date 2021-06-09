package instancemetadata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
)

type InstanceMetadata interface {
	Hostname() string
	TenantID() string
	SubscriptionID() string
	Location() string
	ResourceGroup() string
	Environment() *azure.Environment
}

type instanceMetadata struct {
	hostname       string
	tenantID       string
	subscriptionID string
	location       string
	resourceGroup  string
	environment    *azure.Environment
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

func (im *instanceMetadata) Environment() *azure.Environment {
	return im.environment
}

func New(ctx context.Context, log *logrus.Entry, isLocalDevelopmentMode bool) (InstanceMetadata, error) {
	if isLocalDevelopmentMode {
		log.Info("creating development InstanceMetadata")
		return NewDev(true)
	}

	if os.Getenv("AZURE_EV2") != "" {
		log.Info("creating InstanceMetadata from Environment")
		return newProdFromEnv(ctx)
	} else {
		log.Info("creating InstanceMetadata from Azure Instance Metadata Service (AIMS)")
		return newProd(ctx)
	}
}
