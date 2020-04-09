package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
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

	b, err := f._getAdminKubernetesObjects(ctx, r)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminKubernetesObjects(ctx context.Context, r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)

	err := validateGetAdminKubernetesObjects(r.URL.Query())
	if err != nil {
		return nil, err
	}

	kind, namespace, name := r.URL.Query().Get("kind"), r.URL.Query().Get("namespace"), r.URL.Query().Get("name")

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.db.OpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	if name != "" {
		return f.kubeActions.Get(ctx, doc.OpenShiftCluster, kind, namespace, name)
	}
	return f.kubeActions.List(ctx, doc.OpenShiftCluster, kind, namespace)
}

// rxKubernetesString is weaker than Kubernetes validation, but strong enough to
// prevent mischief
var rxKubernetesString = regexp.MustCompile(`(?i)^[-a-z0-9]{0,255}$`)

func validateGetAdminKubernetesObjects(q url.Values) error {
	kind := q.Get("kind")
	if kind == "" ||
		!rxKubernetesString.MatchString(kind) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided kind '%s' is invalid.", kind)
	}
	if strings.EqualFold(kind, "secret") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to secrets is forbidden.")
	}

	namespace := q.Get("namespace")
	if !rxKubernetesString.MatchString(namespace) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided namespace '%s' is invalid.", namespace)
	}

	name := q.Get("name")
	if !rxKubernetesString.MatchString(name) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided name '%s' is invalid.", name)
	}

	return nil
}
