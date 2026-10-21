package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	v20250725 "github.com/Azure/ARO-RP/pkg/api/v20250725"
	"github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	utilarmredhatopenshift "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armredhatopenshift"
)

type InternalClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string) (*api.OpenShiftCluster, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters *api.OpenShiftCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type clientCluster interface {
	armredhatopenshift.OpenShiftCluster
}

type apiCluster interface {
	v20250725.OpenShiftCluster
}

type externalClient[ClientCluster clientCluster] interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string, options *armredhatopenshift.OpenShiftClustersClientGetOptions) (armredhatopenshift.OpenShiftClustersClientGetResponse, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters ClientCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type internalClient[ClientCluster clientCluster, ApiCluster apiCluster] struct {
	externalClient externalClient[ClientCluster]
	converter      api.OpenShiftClusterConverter
}

func NewInternalClient(log *logrus.Entry, environment env.Core, authorizer autorest.Authorizer) (InternalClient, error) {
	log.Infof("Using ARO API version [%s]", v20250725.APIVersion)
	options := environment.Environment().EnvironmentCredentialOptions()
	spTokenCredential, err := azidentity.NewEnvironmentCredential(options)
	if err != nil {
		return nil, err
	}

	externalClient, err := utilarmredhatopenshift.NewOpenShiftClustersClient(environment.SubscriptionID(), spTokenCredential, environment.ArmClientOptions())
	if err != nil {
		return nil, err
	}

	return &internalClient[armredhatopenshift.OpenShiftCluster, v20250725.OpenShiftCluster]{
		externalClient: externalClient,
		converter:      api.APIs[v20250725.APIVersion].OpenShiftClusterConverter,
	}, nil
}

func (c *internalClient[ClientCluster, ApiCluster]) Get(ctx context.Context, resourceGroupName string, resourceName string) (*api.OpenShiftCluster, error) {
	ocExt, err := c.externalClient.Get(ctx, resourceGroupName, resourceName, nil)
	if err != nil {
		return nil, err
	}

	found := ClientCluster(ocExt.OpenShiftCluster)
	return c.toInternal(&found)
}

func (c *internalClient[ClientCluster, ApiCluster]) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters *api.OpenShiftCluster) error {
	ocExt, err := c.toExternal(parameters)
	if err != nil {
		return err
	}

	return c.externalClient.CreateOrUpdateAndWait(ctx, resourceGroupName, resourceName, *ocExt)
}

func (c *internalClient[ClientCluster, ApiCluster]) DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error {
	return c.externalClient.DeleteAndWait(ctx, resourceGroupName, resourceName)
}

// We use JSON marshaling/unmarshaling to convert between our "external/versioned" cluster struct in pkg/api,
// and the struct in the generated clients
func (c *internalClient[ClientCluster, ApiCluster]) toExternal(oc *api.OpenShiftCluster) (*ClientCluster, error) {
	apiExt := c.converter.ToExternal(oc)
	ocExt := new(ClientCluster)

	data, err := json.Marshal(apiExt)
	if err != nil {
		return ocExt, err
	}

	err = json.Unmarshal(data, &ocExt)
	return ocExt, err
}

func (c *internalClient[ClientCluster, ApiCluster]) toInternal(ocExt *ClientCluster) (*api.OpenShiftCluster, error) {
	oc := &api.OpenShiftCluster{}
	apiExt := new(ApiCluster)

	data, err := json.Marshal(ocExt)
	if err != nil {
		return oc, err
	}

	err = json.Unmarshal(data, apiExt)
	if err != nil {
		return oc, err
	}
	c.converter.ToInternal(apiExt, oc)
	return oc, nil
}
