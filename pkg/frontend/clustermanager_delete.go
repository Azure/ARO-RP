package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) deleteClusterManagerConfiguration(w http.ResponseWriter, r *http.Request) {
	reply(r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry), w, nil, []byte("delete request"), nil)
}
