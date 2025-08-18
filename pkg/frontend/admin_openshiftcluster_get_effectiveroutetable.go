package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenshiftClusterEffectiveRouteTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	// Use filepath.Dir to get the cluster resource path (same as original)
	r.URL.Path = filepath.Dir(r.URL.Path)

	e, err := f._getOpenshiftClusterEffectiveRouteTable(ctx, r)

	adminReply(log, w, nil, e, err)
}
