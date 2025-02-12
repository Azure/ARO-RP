package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

var csrResource = schema.GroupVersionResource{
	Group:    "certificates.k8s.io",
	Resource: "certificatesigningrequests",
}

func (f *frontend) postAdminOpenShiftClusterApproveCSR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminOpenShiftClusterApproveCSR(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterApproveCSR(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	csrName := r.URL.Query().Get("csrName")
	if csrName != "" {
		err := validateAdminKubernetesObjects(r.Method, csrResource, "", csrName)
		if err != nil {
			return err
		}
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	if csrName != "" {
		return k.ApproveCsr(ctx, csrName)
	}

	return k.ApproveAllCsrs(ctx)
}
