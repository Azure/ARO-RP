package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOperations(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)
	b, err := json.MarshalIndent(api.AllOperations, "", "    ")
	reply(log, w, nil, b, err)
}
