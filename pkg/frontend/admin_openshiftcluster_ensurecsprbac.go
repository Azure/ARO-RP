package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenshiftClusterEnsureCSPRBAC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	err := f._postAdminOpenshiftClusterEnsureCSPRBAC(ctx, log, resourceID)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenshiftClusterEnsureCSPRBAC(ctx context.Context, log *logrus.Entry, resourceID string) error {
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return err
	}
	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			fmt.Sprintf("The Resource %s was not found.", resourceID))
	case err != nil:
		return err
	}
	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return err
	}
	azureActions, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return err
	}
	return azureActions.EnsureClusterServicePrincipalRBAC(ctx)
}
