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

	resource := chi.URLParam(r, "resource")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	if resource == "" {
		adminReply(
			log, w, nil, nil,
			api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidRequestContent,
				"",
				"resource is required",
			),
		)
		return
	}

	if namespace == "" {
		adminReply(
			log, w, nil, nil,
			api.NewCloudError(
				http.StatusBadRequest,
				api.CloudErrorCodeInvalidRequestContent,
				"",
				"namespace is required",
			),
		)
		return
	}

	if f.hiveK8sObjectManager == nil {
		adminReply(
			log, w, nil, nil,
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

	if name != "" {
		// GET path
		b, err = f.hiveK8sObjectManager.Get(ctx, resource, namespace, name)
	} else {
		// LIST path — namespace is handled inside kubeActions
		b, err = f.hiveK8sObjectManager.List(ctx, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}
