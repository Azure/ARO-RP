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

func (f *frontend) postAdminOpenShiftClusterApproveCSR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminOpenShiftClusterApproveCSR(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterApproveCSR(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	vars := mux.Vars(r)

	csrName := r.URL.Query().Get("csrName")
	if csrName != "" {
		err := validateAdminKubernetesObjects(r.Method, "CertificateSigningRequest", "", csrName)
		if err != nil {
			return err
		}
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
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
