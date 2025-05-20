package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// getAdminTopPods retrieves the top pod metrics and sends the response.
func (f *frontend) getAdminTopPods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	resourceID = strings.TrimSuffix(resourceID, "/top/pods")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		msg := fmt.Sprintf("Failed to access OpenShiftClusters DB: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		msg := fmt.Sprintf("OpenShiftCluster resource %q not found in DB", resourceID)
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", msg)
		return
	}

	restConfig, err := restconfig.RestConfig(f.env, doc.OpenShiftCluster)
	if err != nil {
		log.WithError(err).Error("failed to create restConfig")
		msg := fmt.Sprintf("Failed to create restConfig: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	ka, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		msg := fmt.Sprintf("Failed to create kubeActions: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	// Parse allNamespaces query param (default: true)
	allNamespaces := true
	if nsFlag := r.URL.Query().Get("allNamespaces"); nsFlag == "false" {
		allNamespaces = false
	}

	result, err := ka.TopPods(ctx, restConfig, allNamespaces)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve pod metrics: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	replyJSON(w, http.StatusOK, result)
}

// getAdminTopNodes retrieves the top node metrics and sends the response.
func (f *frontend) getAdminTopNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")
	resourceID = strings.TrimSuffix(resourceID, "/top/nodes")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		msg := fmt.Sprintf("Failed to access OpenShiftClusters DB: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		msg := fmt.Sprintf("Resource not found: %v", err)
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", msg)
		return
	}

	restConfig, err := restconfig.RestConfig(f.env, doc.OpenShiftCluster)
	if err != nil {
		log.WithError(err).Error("failed to create restConfig")
		msg := fmt.Sprintf("Failed to create restConfig: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	ka, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		msg := fmt.Sprintf("Failed to create kubeActions: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	result, err := ka.TopNodes(ctx, restConfig)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve node metrics: %v", err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", msg)
		return
	}

	replyJSON(w, http.StatusOK, result)
}

// replyJSON sends a JSON response.
func replyJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
