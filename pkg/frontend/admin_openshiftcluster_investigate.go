package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/holmes"
)

type investigateRequest struct {
	Question string `json:"question"`
}

func (f *frontend) postAdminOpenShiftClusterInvestigate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._postAdminOpenShiftClusterInvestigate(ctx, r, log, w)
	if err != nil {
		// Only set Content-Type and call adminReply on error, since on success
		// the response was already streamed as text/plain by InvestigateCluster.
		adminReply(log, w, nil, nil, err)
	}
}

func (f *frontend) _postAdminOpenShiftClusterInvestigate(ctx context.Context, r *http.Request, log *logrus.Entry, w http.ResponseWriter) error {
	resType, resName, resGroupName := chi.URLParam(r, "resourceType"), chi.URLParam(r, "resourceName"), chi.URLParam(r, "resourceGroupName")

	// Parse request body from context (middleware buffers the body).
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	var req investigateRequest
	err := json.Unmarshal(body, &req)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request body could not be parsed: %v.", err))
	}

	if req.Question == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "question", "The question parameter is required and must be non-empty.")
	}

	const maxQuestionLength = 1000
	if len(req.Question) > maxQuestionLength {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "question", fmt.Sprintf("The question must not exceed %d characters.", maxQuestionLength))
	}

	holmesConfig := holmes.NewHolmesConfigFromEnv()

	// Rate limit: reject if too many concurrent investigations are running.
	current := atomic.AddInt64(&f.activeInvestigations, 1)
	defer atomic.AddInt64(&f.activeInvestigations, -1)
	if current > int64(holmesConfig.MaxConcurrentInvestigations) {
		return api.NewCloudError(http.StatusTooManyRequests, api.CloudErrorCodeThrottlingLimitExceeded, "", fmt.Sprintf("Too many concurrent investigations (%d). Please try again later.", holmesConfig.MaxConcurrentInvestigations))
	}

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s' under resource group '%s' was not found.", resType, resName, resGroupName))
	case err != nil:
		return err
	}

	if f.hiveClusterManager == nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}

	hiveNamespace := doc.OpenShiftCluster.Properties.HiveProfile.Namespace
	if hiveNamespace == "" {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "cluster does not have a Hive namespace configured")
	}

	// Generate a short-lived (1h) read-only kubeconfig for the diagnostics identity.
	// This uses the cluster CA from the persisted graph to sign a fresh client cert,
	// then converts to the external API endpoint since Hive cannot resolve api-int.*.
	kubeconfig, err := f.generateDiagnosticsKubeconfig(ctx, log, doc)
	if err != nil {
		return fmt.Errorf("failed to generate diagnostics kubeconfig: %w", err)
	}

	log.Infof("starting Holmes investigation for cluster %s with question: %s", resourceID, req.Question)

	// Set Content-Type before streaming begins. Once bytes are written to w,
	// the response is committed and errors cannot be reported via adminReply.
	w.Header().Set("Content-Type", "text/plain")

	err = f.hiveClusterManager.InvestigateCluster(ctx, hiveNamespace, kubeconfig, holmesConfig, req.Question, w)
	if err != nil {
		return fmt.Errorf("failed to investigate cluster: %w", err)
	}

	return nil
}
