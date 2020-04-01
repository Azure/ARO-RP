package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAsyncOperationResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	header := http.Header{}
	b, err := f._getAsyncOperationResult(ctx, r, header, f.apis[vars["api-version"]].OpenShiftClusterConverter())

	reply(log, w, header, b, err)
}

func (f *frontend) _getAsyncOperationResult(ctx context.Context, r *http.Request, header http.Header, converter api.OpenShiftClusterConverter) ([]byte, error) {
	vars := mux.Vars(r)

	asyncdoc, err := f.db.AsyncOperations.Get(ctx, vars["operationId"])
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The entity was not found.")
	case err != nil:
		return nil, err
	}

	doc, err := f.db.OpenShiftClusters.Get(ctx, asyncdoc.OpenShiftClusterKey)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	// don't give away the final operation status until it's committed to the
	// database
	if doc != nil && doc.AsyncOperationID == vars["operationId"] {
		header["Location"] = r.Header["Referer"]
		return nil, statusCodeError(http.StatusAccepted)
	}

	if asyncdoc.OpenShiftCluster == nil {
		return nil, statusCodeError(http.StatusNoContent)
	}

	asyncdoc.OpenShiftCluster.Properties.ClusterProfile.PullSecret = ""
	asyncdoc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(converter.ToExternal(asyncdoc.OpenShiftCluster), "", "    ")
}
