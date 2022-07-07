package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAsyncOperationsStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	b, err := f._getAsyncOperationsStatus(ctx, r)

	reply(log, w, nil, b, err)
}

func (f *frontend) _getAsyncOperationsStatus(ctx context.Context, r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)

	asyncdoc, err := f.dbAsyncOperations.Get(ctx, vars["operationId"])
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The entity was not found.")
	case err != nil:
		return nil, err
	}

	resource, err := azure.ParseResourceID(asyncdoc.OpenShiftClusterKey)
	switch {
	case err != nil:
		return nil, err
	case resource.SubscriptionID != vars["subscriptionId"]:
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The entity was not found.")
	}

	doc, err := f.dbOpenShiftClusters.Get(ctx, asyncdoc.OpenShiftClusterKey)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	// don't give away the final operation status until it's committed to the
	// database
	if doc != nil && doc.AsyncOperationID == vars["operationId"] {
		asyncdoc.AsyncOperation.ProvisioningState = asyncdoc.AsyncOperation.InitialProvisioningState
		asyncdoc.AsyncOperation.EndTime = nil
		asyncdoc.AsyncOperation.Error = nil
	}

	asyncdoc.AsyncOperation.MissingFields = api.MissingFields{}
	asyncdoc.AsyncOperation.InitialProvisioningState = ""

	h := &codec.JsonHandle{
		Indent: 4,
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, h).Encode(asyncdoc.AsyncOperation)
	if err != nil {
		return nil, err
	}

	return b, nil
}
