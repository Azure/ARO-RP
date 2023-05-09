package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
)

// Dynamic validate in the operator context.
type Dynamic interface {
	ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidateEncryptionAtHost(ctx context.Context, oc *api.OpenShiftCluster) error
}

type dynamic struct {
	log            *logrus.Entry
	authorizerType AuthorizerType
	env            env.Interface
	azEnv          *azureclient.AROEnvironment

	diskEncryptionSets   compute.DiskEncryptionSetsClient
	resourceSkusClient   compute.ResourceSkusClient
	permissionsValidator permissions.PermissionsValidator
}

type AuthorizerType string

const (
	AuthorizerFirstParty              AuthorizerType = "resource provider"
	AuthorizerClusterServicePrincipal AuthorizerType = "cluster"
)

func NewValidator(
	log *logrus.Entry,
	env env.Interface,
	azEnv *azureclient.AROEnvironment,
	subscriptionID string,
	authorizer autorest.Authorizer,
	appID string,
	authorizerType AuthorizerType,
	permissionsValidator permissions.PermissionsValidator,
) Dynamic {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		env:            env,
		azEnv:          azEnv,

		diskEncryptionSets:   compute.NewDiskEncryptionSetsClient(azEnv, subscriptionID, authorizer),
		resourceSkusClient:   compute.NewResourceSkusClient(azEnv, subscriptionID, authorizer),
		permissionsValidator: permissionsValidator,
	}
}
