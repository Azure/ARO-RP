package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

type azVmActionsWrapper struct {
	vmName string
	doc    *api.OpenShiftClusterDocument
}

func (f *frontend) newAzureActionsWrapper(log *logrus.Entry, ctx context.Context, vmName, resourceID string, vars map[string]string) (azVmActionsWrapper, error) {
	err := validateAdminVMName(vmName)
	if err != nil {
		return azVmActionsWrapper{}, err
	}
	if err != nil {
		return azVmActionsWrapper{}, err
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return azVmActionsWrapper{}, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return azVmActionsWrapper{}, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return azVmActionsWrapper{}, err
	}

	f.adminAction, err = f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return azVmActionsWrapper{}, err
	}
	return azVmActionsWrapper{
		vmName: vmName,
		doc:    doc,
	}, nil
}
