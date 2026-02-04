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

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterStopVM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	err := f._postAdminOpenShiftClusterStopVM(log, ctx, r)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterStopVM(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	vmName, deallocateVm := r.URL.Query().Get("vmName"), r.URL.Query().Get("deallocateVM")
	resourceName := chi.URLParam(r, "resourceName")
	resourceType := chi.URLParam(r, "resourceType")
	resourceGroupName := chi.URLParam(r, "resourceGroupName")

	action, _, err := f.prepareAdminActions(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), resourceType, resourceName, resourceGroupName)
	if err != nil {
		return err
	}

	return action.VMStopAndWait(ctx, vmName, strings.EqualFold(deallocateVm, "True"))
}
