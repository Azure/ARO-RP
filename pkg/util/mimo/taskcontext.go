package mimo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerregistry"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

type TaskContext interface {
	context.Context
	Now() time.Time
	Environment() env.Interface
	Log() *logrus.Entry

	// Result messages
	SetResultMessage(string)

	// OpenShiftCluster
	GetClusterUUID() string
	GetOpenShiftClusterProperties() api.OpenShiftClusterProperties
	GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument

	// Kubernetes client
	ClientHelper() (clienthelper.Interface, error)

	// All Azure clients that MIMO tasks interact with _must_ be Track 2 SDK
	// clients. If you need something with only a Track 1 client in pkg/util/,
	// first port it to be Track 2 before including it here.

	// Azure Networking clients
	InterfacesClient() (armnetwork.InterfacesClient, error)
	LoadBalancersClient() (armnetwork.LoadBalancersClient, error)
	PrivateLinkServicesClient() (armnetwork.PrivateLinkServicesClient, error)

	// Azure Compute clients
	ResourceSKUsClient() (armcompute.ResourceSKUsClient, error)

	// Azure Container Registry clients
	TokensClient() (armcontainerregistry.TokensClient, error)
	RegistriesClient() (armcontainerregistry.RegistriesClient, error)
}

func GetTaskContext(c context.Context) (TaskContext, error) {
	r, ok := c.(TaskContext)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v to TaskContext", c)
	}

	return r, nil
}
