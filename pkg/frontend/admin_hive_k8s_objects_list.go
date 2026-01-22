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
	name := r.URL.Query().Get("name")

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

	// Manager must be wired (tests mock this)
	if f.hiveK8sObjectManager == nil {
		adminReply(
			log,
			w,
			nil,
			nil,
			api.NewCloudError(
				http.StatusNotImplemented,
				api.CloudErrorCodeInternalServerError,
				"",
				"hive k8s object manager not configured",
			),
		)
		return
	}

	var (
		b   []byte
		err error
	)

	// Delegate to manager
	if name != "" {
		b, err = f.hiveK8sObjectManager.Get(ctx, region, resource, name)
	} else {
		b, err = f.hiveK8sObjectManager.List(ctx, region, resource)
	}

	adminReply(log, w, nil, b, err)
}
