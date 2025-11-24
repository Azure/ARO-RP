package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
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

	if !nodeIsMaster(vmName) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "",
			fmt.Sprintf(
				`"The vmName '%s' provided cannot be resized. It is either not a master node or not adhering to the standard naming convention."`,
				vmName))
	}

	action, _, err := f.prepareAdminActions(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), resourceType, resourceName, resourceGroupName)
	if err != nil {
		return err
	}

	vmSize := r.URL.Query().Get("vmSize")
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

	return action.VMResize(ctx, vmName, vmSize)
}

// Check if VM name follows standard control plane node naming convention
// vmName is expected to end with pattern "-master-[0-9]" or "-master-[0-9A-Za-z]{5}-[0-9]" for nodes created via CPMS
func nodeIsMaster(vmName string) bool {
	r := regexp.MustCompile(`.*-master-([0-9A-Za-z]{5}-)?[0-9]$`)
	return r.MatchString(vmName)
}
