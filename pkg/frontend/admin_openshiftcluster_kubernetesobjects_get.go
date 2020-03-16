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

func (f *frontend) getAdminKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	jb, err := f._getAdminKubernetesObjects(ctx, r)

	reply(log, w, nil, jb, err)
}

func (f *frontend) _getAdminKubernetesObjects(ctx context.Context, r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)

	kind := r.URL.Query().Get("kind")
	if kind == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The request query was invalid: %q.", "kind is required")
	}
	if strings.EqualFold(kind, "secret") {
		return nil, api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to Secrets are forbidden: %q.", "")
	}
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	if namespace != "" && name != "" {
		return f.kubeActions.Get(ctx, doc.OpenShiftCluster, kind, namespace, name)
	}
	return f.kubeActions.List(ctx, doc.OpenShiftCluster, kind, namespace)
}
