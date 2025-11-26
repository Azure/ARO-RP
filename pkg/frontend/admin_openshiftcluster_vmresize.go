package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"github.com/ugorji/go/codec"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	aromachine "github.com/Azure/ARO-RP/pkg/util/machine"
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

	action, oc, err := f.prepareAdminActions(log, ctx, vmName, strings.TrimPrefix(r.URL.Path, "/admin"), resourceType, resourceName, resourceGroupName)
	if err != nil {
		return err
	}

	k, err := f.kubeActionsFactory(log, f.env, oc.OpenShiftCluster)
	if err != nil {
		return err
	}

	rawMachine, err := k.KubeGet(ctx, "machine", "openshift-machine-api", vmName)
	if err != nil {
		return err
	}

	machine := &machinev1beta1.Machine{}
	err = codec.NewDecoderBytes(rawMachine, &codec.JsonHandle{}).Decode(machine)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Sprintf("failed to decode machine object for %s, %s", vmName, err.Error()))
	}

	isControlPlaneMachine, err := aromachine.IsMasterRole(machine)
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	if !isControlPlaneMachine {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "",
			fmt.Sprintf(`"The vmName '%s' provided cannot be resized. It is not a control plane node."`, vmName))
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
