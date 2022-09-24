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

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterStartVM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	err := f._postAdminOpenShiftClusterStartVM(log, ctx, r)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterStartVM(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	vars := mux.Vars(r)
	vmName := r.URL.Query().Get("vmName")
	azActionsWrapper, err := f.newAzureActionsWrapper(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), vars)
	if err != nil {
		return err
	}

	return f.adminAction.VMStartAndWait(ctx, azActionsWrapper.vmName)
}
