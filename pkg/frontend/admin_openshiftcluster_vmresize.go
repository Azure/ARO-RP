package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterVMResize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	err := f._postAdminOpenShiftClusterVMResize(log, ctx, r)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterVMResize(log *logrus.Entry, ctx context.Context, r *http.Request) error {
	vmName := r.URL.Query().Get("vmName")
	resourceName := chi.URLParam(r, "resourceName")
	resourceType := chi.URLParam(r, "resourceType")
	resourceGroupName := chi.URLParam(r, "resourceGroupName")
	vmSize := r.URL.Query().Get("vmSize")

	action, _, err := f.prepareAdminActions(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), resourceType, resourceName, resourceGroupName)
	if err != nil {
		return err
	}

	err = validateAdminMasterVMSize(vmSize)
	if err != nil {
		return err
	}

	// checks if the Virtual machines exists in the Cluster RG
	exists, err := action.ResourceGroupHasVM(ctx, vmName)
	if err != nil {
		return err
	}
	if !exists {
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "",
			fmt.Sprintf(
				`"The VirtualMachine '%s' under resource group '%s' was not found."`,
				vmName, resourceGroupName))
	}

	err = action.VMResize(ctx, vmName, vmSize)
	if err != nil {
		// Before sending the error to the resize GA, we'll attempt to power the VM on
		poweronErr := action.VMStartAndWait(ctx, vmName)

		if poweronErr != nil {
			err = errors.Join(err, poweronErr)
		}

		// TO-DO: uncordon the node (requires a kubeclient)
	}
	return err
}
