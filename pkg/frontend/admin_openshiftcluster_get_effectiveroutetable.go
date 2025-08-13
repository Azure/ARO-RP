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
	r.URL.Path = filepath.Dir(r.URL.Path)

	e, err := f._getOpenshiftClusterEffectiveRouteTable(ctx, w, r, log)
	if err != nil {
		log.Fatalf("Unable to get effective route table: %v", err)
	}

	adminReply(log, w, nil, e, err)
}
