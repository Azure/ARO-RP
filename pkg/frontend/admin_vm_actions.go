package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
)

func (f *frontend) prepareAdminActions(log *logrus.Entry, ctx context.Context, vmName, resourceID string, resourceType, resourceName, resourceGroupName string) (azureActions adminactions.AzureActions, doc *api.OpenShiftClusterDocument, err error) {
	err = validateAdminVMName(vmName)
	if err != nil {
		return nil, nil, err
	}

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, nil, err
	}

	doc, err = dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, nil,
			api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
				fmt.Sprintf(
					"The Resource '%s/%s' under resource group '%s' was not found.",
					resourceType, resourceName, resourceGroupName))
	case err != nil:
		return nil, nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, nil, err
	}

	azureActions, err = f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, nil, err
	}
	return azureActions, doc, err
}
