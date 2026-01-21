package mimo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcompute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

type TaskContext interface {
	context.Context
	Now() time.Time
	Environment() env.Interface
	ClientHelper() (clienthelper.Interface, error)
	Log() *logrus.Entry
	LocalFpAuthorizer() (autorest.Authorizer, error)

	// OpenShiftCluster
	GetClusterUUID() string
	GetOpenShiftClusterProperties() api.OpenShiftClusterProperties
	GetOpenshiftClusterDocument() *api.OpenShiftClusterDocument
	// PatchOpenShiftClusterDocument requires an active lease, and only works for the present document
	PatchOpenShiftClusterDocument(context.Context, database.OpenShiftClusterDocumentMutator) (*api.OpenShiftClusterDocument, error)

	// Subscription
	GetTenantID() string

	SetResultMessage(string)
	GetResultMessage() string
}

type TaskContextWithAzureClients interface {
	LoadBalancersClient() armnetwork.LoadBalancersClient
	PrivateLinkServicesClient() armnetwork.PrivateLinkServicesClient
	ResourceSkusClient() armcompute.ResourceSKUsClient
}

func GetTaskContext(c context.Context) (TaskContext, error) {
	r, ok := c.(TaskContext)
	if !ok {
		return nil, fmt.Errorf("cannot convert %v", r)
	}

	return r, nil
}
