package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) topAdminOpenShiftPods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._topAdminOpenShiftPods(ctx, r, log)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _topAdminOpenShiftPods(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	vars := mux.Vars(r)

	namespace := r.URL.Query().Get("namespace")

	err := validateNamespace(r.Method, namespace)
	if err != nil {
		return nil, err
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	m, err := f.metricActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return nil, err
	}

	return m.TopPods(ctx, namespace)
}
