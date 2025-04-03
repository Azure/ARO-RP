package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// getAdminTopPods retrieves the top pod metrics and sends the response.
func (f *frontend) getAdminTopPods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		replyInternalServerError(w)
		return
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	// 🔁 Create restConfig on demand
	restConfig, err := restconfig.RestConfig(f.env, nil)
	if err != nil {
		log.WithError(err).Error("failed to create restConfig")
		replyInternalServerError(w)
		return
	}

	ka, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	result, err := ka.TopPods(ctx, restConfig, true)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	replyJSON(w, http.StatusOK, result)
}

// getAdminTopNodes retrieves the top node metrics and sends the response.
func (f *frontend) getAdminTopNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		replyInternalServerError(w)
		return
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	// 🔁 Create restConfig on demand
	restConfig, err := restconfig.RestConfig(f.env, nil)
	if err != nil {
		log.WithError(err).Error("failed to create restConfig")
		replyInternalServerError(w)
		return
	}

	ka, err := f.kubeActionsFactory(log, f.env, doc.OpenShiftCluster)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	result, err := ka.TopNodes(ctx, restConfig)
	if err != nil {
		replyInternalServerError(w)
		return
	}

	replyJSON(w, http.StatusOK, result)
}

// fallback helpers if util not imported
func replyInternalServerError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func replyJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
