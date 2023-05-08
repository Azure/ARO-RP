package dynamic

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/permissions"
)

type Subnet struct {
	// ID is a resource id of the subnet
	ID string

	// Path is a path in the cluster document. For example, properties.workerProfiles[0].subnetId
	Path string
}

type ServicePrincipalValidator interface {
	ValidateServicePrincipal(ctx context.Context, tokenCredential azcore.TokenCredential) error
}

type VNetValidator interface {
	ValidateVnet(ctx context.Context, location string, subnets []Subnet, additionalCIDRs ...string) error
}

// Dynamic validate in the operator context.
type Dynamic interface {
	ServicePrincipalValidator
	VNetValidator

	ValidateSubnets(ctx context.Context, oc *api.OpenShiftCluster, subnets []Subnet) error
	ValidateDiskEncryptionSets(ctx context.Context, oc *api.OpenShiftCluster) error
	ValidateEncryptionAtHost(ctx context.Context, oc *api.OpenShiftCluster) error
}

type dynamic struct {
	log            *logrus.Entry
	appID          string // for use when reporting an error
	authorizerType AuthorizerType
	env            env.Interface
	azEnv          *azureclient.AROEnvironment

	permissions        authorization.PermissionsClient
	virtualNetworks    virtualNetworksGetClient
	diskEncryptionSets compute.DiskEncryptionSetsClient
	resourceSkusClient compute.ResourceSkusClient
	spComputeUsage     compute.UsageClient
	spNetworkUsage     network.UsageClient
	pdpChecker         *PDPChecker
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
	pdpChecker *PDPChecker,
) Dynamic {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		env:            env,
		azEnv:          azEnv,

		spComputeUsage: compute.NewUsageClient(azEnv, subscriptionID, authorizer),
		spNetworkUsage: network.NewUsageClient(azEnv, subscriptionID, authorizer),
		permissions:    authorization.NewPermissionsClient(azEnv, subscriptionID, authorizer),
		virtualNetworks: newVirtualNetworksCache(
			network.NewVirtualNetworksClient(azEnv, subscriptionID, authorizer),
		),
		diskEncryptionSets: compute.NewDiskEncryptionSetsClient(azEnv, subscriptionID, authorizer),
		resourceSkusClient: compute.NewResourceSkusClient(azEnv, subscriptionID, authorizer),
		pdpChecker:         pdpChecker,
	}
}

func NewServicePrincipalValidator(
	log *logrus.Entry,
	azEnv *azureclient.AROEnvironment,
	authorizerType AuthorizerType,
) ServicePrincipalValidator {
	return &dynamic{
		log:            log,
		authorizerType: authorizerType,
		azEnv:          azEnv,
	}
}

func (dv *dynamic) validateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	c := closure{dv: dv, ctx: ctx, resource: r, actions: actions}
	conditionalFunc := c.usingListPermissions
	timeout := 20 * time.Second
	if dv.pdpChecker != nil {
		conditionalFunc = c.usingCheckAccessV2
		timeout = 65 * time.Second // checkAccess refreshes data every min. This allows ~3 retries.
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollImmediateUntil(timeout, conditionalFunc, timeoutCtx.Done())
}

// closure is the closure used in PollImmediateUntil's ConditionalFunc
type closure struct {
	dv       *dynamic
	ctx      context.Context
	resource *azure.Resource
	actions  []string
	oid      *string
}

// usingListPermissions is how the current check is done
func (c closure) usingListPermissions() (bool, error) {
	c.dv.log.Debug("retry validateActions with ListPermissions")
	perms, err := c.dv.permissions.ListForResource(
		c.ctx,
		c.resource.ResourceGroup,
		c.resource.Provider,
		"",
		c.resource.ResourceType,
		c.resource.ResourceName,
	)
	if err != nil {
		return false, err
	}

	for _, action := range c.actions {
		ok, err := permissions.CanDoAction(perms, action)
		if !ok || err != nil {
			// TODO(jminter): I don't understand if there are genuinely
			// cases where CanDoAction can return false then true shortly
			// after. I'm a little skeptical; if it can't happen we can
			// simplify this code.  We should add a metric on this.
			return false, err
		}
	}
	return true, nil
}
