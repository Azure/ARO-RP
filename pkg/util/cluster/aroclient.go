package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	v20230904 "github.com/Azure/ARO-RP/pkg/api/v20230904"
	v20240812preview "github.com/Azure/ARO-RP/pkg/api/v20240812preview"
	mgmtredhatopenshift20230904 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2023-09-04/redhatopenshift"
	mgmtredhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2024-08-12-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/env"
	redhatopenshift20230904 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2023-09-04/redhatopenshift"
	redhatopenshift20240812preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2024-08-12-preview/redhatopenshift"
)

type InternalClient interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string) (*api.OpenShiftCluster, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters *api.OpenShiftCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type clientCluster interface {
	mgmtredhatopenshift20230904.OpenShiftCluster | mgmtredhatopenshift20240812preview.OpenShiftCluster
}

type apiCluster interface {
	v20230904.OpenShiftCluster | v20240812preview.OpenShiftCluster
}

type externalClient[ClientCluster clientCluster] interface {
	Get(ctx context.Context, resourceGroupName string, resourceName string) (ClientCluster, error)
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, resourceName string, parameters ClientCluster) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error
}

type internalClient[ClientCluster clientCluster, ApiCluster apiCluster] struct {
	externalClient   externalClient[ClientCluster]
	converter        api.OpenShiftClusterConverter
	newClientCluster func() ClientCluster
	newApiCluster    func() ApiCluster
}

func NewInternalClient(log *logrus.Entry, environment env.Core, authorizer autorest.Authorizer) InternalClient {
	if env.IsLocalDevelopmentMode() {
		log.Infof("Using ARO API version [%s]", v20240812preview.APIVersion)
		return &internalClient[mgmtredhatopenshift20240812preview.OpenShiftCluster, v20240812preview.OpenShiftCluster]{
			externalClient: redhatopenshift20240812preview.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
			converter:      api.APIs[v20240812preview.APIVersion].OpenShiftClusterConverter,
			newClientCluster: func() mgmtredhatopenshift20240812preview.OpenShiftCluster {
				return mgmtredhatopenshift20240812preview.OpenShiftCluster{}
			},
			newApiCluster: func() v20240812preview.OpenShiftCluster {
				return v20240812preview.OpenShiftCluster{}
			},
		}
	}

	log.Infof("Using ARO API version [%s]", v20230904.APIVersion)
	return &internalClient[mgmtredhatopenshift20230904.OpenShiftCluster, v20230904.OpenShiftCluster]{
		externalClient: redhatopenshift20230904.NewOpenShiftClustersClient(environment.Environment(), environment.SubscriptionID(), authorizer),
		converter:      api.APIs[v20230904.APIVersion].OpenShiftClusterConverter,
		newClientCluster: func() mgmtredhatopenshift20230904.OpenShiftCluster {
			return mgmtredhatopenshift20230904.OpenShiftCluster{}
		},
		newApiCluster: func() v20230904.OpenShiftCluster {
			return v20230904.OpenShiftCluster{}
		},
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

	return c.externalClient.CreateOrUpdateAndWait(ctx, resourceGroupName, resourceName, ocExt)
}

func (c *internalClient[ClientCluster, ApiCluster]) DeleteAndWait(ctx context.Context, resourceGroupName string, resourceName string) error {
	return c.externalClient.DeleteAndWait(ctx, resourceGroupName, resourceName)
}

// We use JSON marshaling/unmarshaling to convert between our "external/versioned" cluster struct in pkg/api,
// and the struct in the generated clients
func (c *internalClient[ClientCluster, ApiCluster]) toExternal(oc *api.OpenShiftCluster) (ClientCluster, error) {
	apiExt := c.converter.ToExternal(oc)
	ocExt := c.newClientCluster()

	data, err := json.Marshal(apiExt)
	if err != nil {
		return ocExt, err
	}

	err = json.Unmarshal(data, &ocExt)
	return ocExt, err
}

func (c *internalClient[ClientCluster, ApiCluster]) toInternal(ocExt *ClientCluster) (*api.OpenShiftCluster, error) {
	oc := &api.OpenShiftCluster{}
	apiExt := c.newApiCluster()

	data, err := json.Marshal(ocExt)
	if err != nil {
		return oc, err
	}

	err = json.Unmarshal(data, &apiExt)
	if err != nil {
		return oc, err
	}
	c.converter.ToInternal(&apiExt, oc)
	return oc, nil
}
