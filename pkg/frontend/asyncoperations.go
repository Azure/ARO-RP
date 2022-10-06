package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (f *frontend) newAsyncOperation(ctx context.Context, vars map[string]string, doc *api.OpenShiftClusterDocument) (string, error) {
	id := f.dbAsyncOperations.NewUUID()
	_, err := f.dbAsyncOperations.Create(ctx, &api.AsyncOperationDocument{
		ID:                  id,
		OpenShiftClusterKey: doc.Key,
		AsyncOperation: &api.AsyncOperation{
			ID:                       f.operationsPath(vars, id),
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

func (f *frontend) operationsPath(vars map[string]string, id string) string {
	return "/subscriptions/" + vars["subscriptionId"] + "/providers/" + vars["resourceProviderNamespace"] + "/locations/" + strings.ToLower(f.env.Location()) + "/operationsstatus/" + id
}

func (f *frontend) operationResultsPath(r *http.Request, id string) string {
	vars := mux.Vars(r)

	return "/subscriptions/" + vars["subscriptionId"] + "/providers/" + vars["resourceProviderNamespace"] + "/locations/" + strings.ToLower(f.env.Location()) + "/operationresults/" + id
}
