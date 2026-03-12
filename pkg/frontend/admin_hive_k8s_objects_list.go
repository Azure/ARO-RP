package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) adminHiveK8sObjectsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	resource := chi.URLParam(r, "resource")
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")

	manager := newHiveK8sObjectManager(f.env, f.kubeActionsFactory)

	var (
		b   []byte
		err error
	)

	if name != "" {
		b, err = manager.Get(ctx, resource, namespace, name)
	} else {
		b, err = manager.List(ctx, resource, namespace)
	}

	adminReply(log, w, nil, b, err)
}
