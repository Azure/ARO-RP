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
)

type investigateRequest struct {
	Question string `json:"question"`
}

// trackingResponseWriter wraps http.ResponseWriter to track whether any bytes
// have been written. This is used to avoid calling adminReply (which writes
// JSON) after streaming has already started (which writes text/plain).
type trackingResponseWriter struct {
	http.ResponseWriter
	written int64
}

func (tw *trackingResponseWriter) Write(b []byte) (int, error) {
	n, err := tw.ResponseWriter.Write(b)
	atomic.AddInt64(&tw.written, int64(n))
	return n, err
}

func (tw *trackingResponseWriter) Flush() {
	if flusher, ok := tw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (f *frontend) postAdminOpenShiftClusterInvestigate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	tw := &trackingResponseWriter{ResponseWriter: w}
	err := f._postAdminOpenShiftClusterInvestigate(ctx, r, log, tw)
	if err != nil {
		if atomic.LoadInt64(&tw.written) > 0 {
			// Streaming already started — can't send a JSON error response.
			// Log the error server-side instead.
			log.WithError(err).Warn("investigation failed after streaming started")
			return
		}
		adminReply(log, tw, nil, nil, err)
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

	// Reject control characters that could affect CLI argument parsing.
	for _, ch := range req.Question {
		if ch < 0x20 && ch != ' ' {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "question", "The question must not contain control characters.")
		}
	}

	if f.holmesConfig == nil {
		return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Holmes investigation is not configured")
	}

	// Rate limit: reject if too many concurrent investigations are running.
	// Use CAS loop so rejected requests don't temporarily inflate the counter.
	// NOTE: This limit is per-RP-instance (in-memory atomic counter). With N
	// replicas, the effective global limit is N * MaxConcurrentInvestigations.
	// A distributed limiter (e.g., CosmosDB-backed) can be added if global
	// quota enforcement is needed.
	maxConcurrent := int64(f.holmesConfig.MaxConcurrentInvestigations)
	for {
		current := atomic.LoadInt64(&f.activeInvestigations)
		if current >= maxConcurrent {
			return api.NewCloudError(http.StatusTooManyRequests, api.CloudErrorCodeThrottlingLimitExceeded, "", fmt.Sprintf("Too many concurrent investigations (%d). Please try again later.", f.holmesConfig.MaxConcurrentInvestigations))
		}
		if atomic.CompareAndSwapInt64(&f.activeInvestigations, current, current+1) {
			break
		}
	}
	defer atomic.AddInt64(&f.activeInvestigations, -1)

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
	// This uses the cluster CA from the persisted graph to sign a fresh client cert.
	// In development mode, the endpoint is rewritten from api-int.* to api.* since
	// the Hive cluster cannot resolve private DNS there.
	kubeconfig, err := f.generateDiagnosticsKubeconfig(ctx, log, doc)
	if err != nil {
		return fmt.Errorf("failed to generate diagnostics kubeconfig: %w", err)
	}

	log.Infof("starting Holmes investigation for cluster %s (question_length=%d)", resourceID, len(req.Question))

	// Set Content-Type before streaming begins. Once bytes are written to w,
	// the response is committed and errors cannot be reported via adminReply.
	w.Header().Set("Content-Type", "text/plain")

	err = f.hiveClusterManager.InvestigateCluster(ctx, hiveNamespace, kubeconfig, f.holmesConfig, req.Question, w)
	if err != nil {
		return fmt.Errorf("failed to investigate cluster: %w", err)
	}

	return nil
}
