package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) listAdminOpenShiftClusterResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	reader, writer := io.Pipe()
	err := f._listAdminOpenShiftClusterResources(ctx, r, writer, log)

	f.streamResponder.AdminReplyStream(log, w, nil, reader, err)
}

func (f *frontend) _listAdminOpenShiftClusterResources(
	ctx context.Context, r *http.Request, writer io.WriteCloser, log *logrus.Entry) error {
	a, err := f.newStreamAzureAction(ctx, r, log)
	if err != nil {
		return err
	}
	return a.WriteToStream(ctx, writer)
}

func (f *frontend) newStreamAzureAction(ctx context.Context, r *http.Request, log *logrus.Entry) (adminactions.AzureActions, error) {
	vars := mux.Vars(r)
	resType, resName, resGroupName := vars["resourceType"], vars["resourceName"], vars["resourceGroupName"]
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found.",
			resType, resName, resGroupName)
	case err != nil:
		return nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	return a, err
}
