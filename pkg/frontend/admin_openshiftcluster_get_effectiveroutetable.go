package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

// getAdminOpenshiftClusterEffectiveRouteTable is the refactored admin version using ARO-RP utilities
func (f *frontend) getAdminOpenshiftClusterEffectiveRouteTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Use filepath.Dir to get the cluster resource path (same as original)
	r.URL.Path = filepath.Dir(r.URL.Path)

	e, err := f._getOpenshiftClusterEffectiveRouteTable(ctx, r)
	if err != nil {
		log.Errorf("Unable to get effective route table: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Failed to retrieve effective route table.")
		return
	}

	adminReply(log, w, nil, e, err)
}
