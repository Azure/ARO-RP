package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) adminHiveK8sObjectsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	region := chi.URLParam(r, "region")
	resource := chi.URLParam(r, "resource")
	namespace := r.URL.Query().Get("namespace")

	// Validate required params
	if region == "" || resource == "" {
		adminReply(
			log,
			w,
			nil,
			nil,
			api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidRequestContent,
				"",
				"region and resource are required",
			),
		)
		return
	}

	// Ensure manager is wired
	if f.hiveK8sObjectManager == nil {
		adminReply(
			log,
			w,
			nil,
			nil,
			api.NewCloudError(
				http.StatusInternalServerError,
				api.CloudErrorCodeInternalServerError,
				"",
				"hive k8s object manager not configured",
			),
		)
		return
	}

	// Delegate to manager
	b, err := f.hiveK8sObjectManager.List(ctx, region, namespace, resource)
	adminReply(log, w, nil, b, err)
}
