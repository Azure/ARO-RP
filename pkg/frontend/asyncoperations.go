package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (f *frontend) newAsyncOperation(ctx context.Context, vars map[string]string, doc *api.OpenShiftClusterDocument) (string, error) {
	id := f.dbAsyncOperations.NewUUID()
	subId := vars["subscriptionId"]
	resourceProviderNamespace := vars["resourceProviderNamespace"]
	_, err := f.dbAsyncOperations.Create(ctx, &api.AsyncOperationDocument{
		ID:                  id,
		OpenShiftClusterKey: doc.Key,
		AsyncOperation: &api.AsyncOperation{
			ID:                       f.operationsPath(subId, resourceProviderNamespace, id),
			Name:                     id,
			InitialProvisioningState: doc.OpenShiftCluster.Properties.ProvisioningState,
			ProvisioningState:        doc.OpenShiftCluster.Properties.ProvisioningState,
			StartTime:                time.Now().UTC(),
		},
	})
	if err != nil {
		return "", err
	}

	return id, nil
}

func (f *frontend) operationsPath(subId, resProviderNamespace, id string) string {
	return "/subscriptions/" + subId + "/providers/" + resProviderNamespace + "/locations/" + strings.ToLower(f.env.Location()) + "/operationsstatus/" + id
}

func (f *frontend) operationResultsPath(subId, resourceProviderNamespace, id string) string {
	return "/subscriptions/" + subId + "/providers/" + resourceProviderNamespace + "/locations/" + strings.ToLower(f.env.Location()) + "/operationresults/" + id
}
