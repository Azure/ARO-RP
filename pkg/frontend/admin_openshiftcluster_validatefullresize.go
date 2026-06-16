package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

// convertErrorLineEndings converts newlines to a clearer separator " | ", as it seems that the new lines are not being parsed in GA
// (or we need to do deeper changes to have nicer error messages)
func convertErrorLineEndings(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	errMsg = strings.ReplaceAll(errMsg, "\n", " | ")
	return fmt.Errorf("%s", errMsg)
}

// validateLiveControlPlaneInventory validates the current Machine, Azure VM, and
// Node inventory for the control plane before or after resize operations.
func validateLiveControlPlaneInventory(
	log *logrus.Entry,
	ctx context.Context,
	kubeActions adminactions.KubeActions,
	azureActions adminactions.AzureActions,
	clusterResourceGroupID string,
) error {
	machines, err := getClusterMachines(ctx, kubeActions)
	if err != nil {
		return err
	}

	ocMachines, err := validateClusterMachines(log, machines)
	if err != nil {
		return fmt.Errorf("control plane machine inventory is inconsistent: %w", err)
	}

	azureVMs, err := getAzureVMs(log, ctx, azureActions, clusterResourceGroupID, ocMachines)
	if err != nil {
		return err
	}

	err = validateClusterMachinesAndVMs(log, ocMachines, azureVMs)
	if err != nil {
		return fmt.Errorf("control plane machine and Azure VM inventory is inconsistent: %w", err)
	}

	ocNodes, err := validateClusterNodes(log, ctx, kubeActions)
	if err != nil {
		return err
	}

	err = validateClusterMachinesAndNodes(log, ocMachines, ocNodes)
	if err != nil {
		return fmt.Errorf("control plane machine and node inventory is inconsistent: %w", err)
	}

	return nil
}

func (f *frontend) getControlPlaneStatusCheckAfterResize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")
	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		apiErr := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		adminReply(log, w, nil, nil, apiErr)
		return
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		apiErr := api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
		adminReply(log, w, nil, nil, apiErr)
		return
	case err != nil:
		adminReply(log, w, nil, nil, err)
		return
	}
	kubeActions, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		apiErr := api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		adminReply(log, w, nil, nil, apiErr)
		return
	}

	azureActions, err := f.newStreamAzureAction(ctx, r, log)
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}
	err = f._getControlPlaneStatusCheckAfterResize(log, ctx, kubeActions, azureActions, doc)
	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _getControlPlaneStatusCheckAfterResize(log *logrus.Entry, ctx context.Context, kubeActions adminactions.KubeActions, azureActions adminactions.AzureActions, doc *api.OpenShiftClusterDocument) error {
	err := validateLiveControlPlaneInventory(log, ctx, kubeActions, azureActions, doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	return convertErrorLineEndings(err)
}
