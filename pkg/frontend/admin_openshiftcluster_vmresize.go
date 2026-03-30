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
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) postAdminOpenShiftClusterVMResize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	vmName := r.URL.Query().Get("vmName")
	resourceName := chi.URLParam(r, "resourceName")
	resourceType := chi.URLParam(r, "resourceType")
	resourceGroupName := chi.URLParam(r, "resourceGroupName")
	vmSize := r.URL.Query().Get("vmSize")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	azureActions, doc, err := f.prepareAdminActions(log, ctx, vmName, resourceID, resourceType, resourceName, resourceGroupName)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	kubeActions, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}
	err = f._postAdminOpenShiftClusterVMResize(log, ctx, kubeActions, azureActions, resourceID, vmName, vmSize, resourceGroupName)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _postAdminOpenShiftClusterVMResize(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions, azureActions adminactions.AzureActions, resourceID string, vmName string, vmSize string, resourceGroupName string) error {
	err := validateAdminMasterVMSize(vmSize)
	if err != nil {
		return err
	}

	// checks if the Virtual machines exists in the Cluster RG
	exists, err := azureActions.ResourceGroupHasVM(ctx, vmName)
	if err != nil {
		return err
	}
	if !exists {
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "",
			fmt.Sprintf(
				`"The VirtualMachine '%s' under resource group '%s' was not found."`,
				vmName, resourceGroupName))
	}

	err = azureActions.VMResize(ctx, vmName, vmSize)
	if err != nil {
		log.Errorf("failed to resize VM '%s' on cluster '%s': %v", vmName, resourceID, err)

		recoveryErr := recoverFromFailedResizeVM(ctx, log, azureActions, kubeActions, vmName, resourceID)
		if recoveryErr != nil {
			return errors.Join(err, recoveryErr)
		}
	}
	return err
}
