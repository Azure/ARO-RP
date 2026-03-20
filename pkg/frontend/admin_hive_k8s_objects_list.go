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

	if f.hiveClusterManager == nil {
		adminReply(log, w, nil, nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled"))
		return
	}

	var (
		b   []byte
		err error
	)

	if name != "" {
		b, err = f.hiveClusterManager.GetHiveK8sObject(ctx, resource, namespace, name)
	} else {
		b, err = f.hiveClusterManager.ListHiveK8sObjects(ctx, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}
