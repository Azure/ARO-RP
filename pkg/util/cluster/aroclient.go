package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	redhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2024-08-12-preview/redhatopenshift"
)

type InternalClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string) (*api.OpenShiftCluster, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters *api.OpenShiftCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type clientCluster interface {
	mgmtredhatopenshift20240812preview.OpenShiftCluster
}

type apiCluster interface {
	v20240812preview.OpenShiftCluster
}

type externalClient[ClientCluster clientCluster] interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string) (ClientCluster, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters ClientCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type internalClient[ClientCluster clientCluster, ApiCluster apiCluster] struct {
	externalClient externalClient[ClientCluster]
	converter      api.OpenShiftClusterConverter
}

func NewInternalClient(log *logrus.Entry, environment env.Core, authorizer autorest.Authorizer) InternalClient {
	log.Infof("Using ARO API version [%s]", v20240812preview.APIVersion)
	return &internalClient[mgmtredhatopenshift20240812preview.OpenShiftCluster, v20240812preview.OpenShiftCluster]{
		externalClient: redhatopenshift20240812preview.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		converter:      api.APIs[v20240812preview.APIVersion].OpenShiftClusterConverter,
	}
}

func (c *internalClient[ClientCluster, ApiCluster]) Get(ctx context.Context, resourceGroupName string, resourceName string) (*api.OpenShiftCluster, error) {
	ocExt, err := c.externalClient.Get(ctx, resourceGroupName, resourceName)
	if err != nil {
		return nil, err
	}

	return c.toInternal(&ocExt)
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
