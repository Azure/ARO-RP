package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminOpenShiftUpgrade(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftUpgrade(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	upgradeY := r.URL.Query().Get("upgradeY") == "true"

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName)
	case err != nil:
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	return k.Upgrade(ctx, upgradeY)
}
