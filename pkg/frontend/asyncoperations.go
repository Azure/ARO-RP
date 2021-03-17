package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	"github.com/Azure/ARO-RP/pkg/api"
)

func (f *frontend) newAsyncOperation(ctx context.Context, r *http.Request, doc *api.OpenShiftClusterDocument) (string, error) {
	newUuid4, _ := uuid.NewV4()
	id := newUuid4.String()
	_, err := f.dbAsyncOperations.Create(ctx, &api.AsyncOperationDocument{
		ID:                  id,
		OpenShiftClusterKey: doc.Key,
		AsyncOperation: &api.AsyncOperation{
			ID:                       f.operationsPath(r, id),
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

func (f *frontend) operationsPath(r *http.Request, id string) string {
	vars := mux.Vars(r)

	return "/subscriptions/" + vars["subscriptionId"] + "/providers/" + vars["resourceProviderNamespace"] + "/locations/" + strings.ToLower(f.env.Location()) + "/operationsstatus/" + id
}

func (f *frontend) operationResultsPath(r *http.Request, id string) string {
	vars := mux.Vars(r)

	return "/subscriptions/" + vars["subscriptionId"] + "/providers/" + vars["resourceProviderNamespace"] + "/locations/" + strings.ToLower(f.env.Location()) + "/operationresults/" + id
}
