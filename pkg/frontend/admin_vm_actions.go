package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
)

func (f *frontend) prepareAdminActions(log *logrus.Entry, ctx context.Context, vmName, resourceID string, vars map[string]string) (azureActions adminactions.AzureActions, doc *api.OpenShiftClusterDocument, err error) {
	err = validateAdminVMName(vmName)
	if err != nil {
		return nil, nil, err
	}

	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	doc, err = f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, nil,
			api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
				"The Resource '%s/%s' under resource group '%s' was not found.",
				resType, resName, resGroupName)
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
