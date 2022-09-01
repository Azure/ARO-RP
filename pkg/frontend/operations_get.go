package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOperations(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	operations := f.apis[vars["api-version"]].OperationList

	b, err := json.MarshalIndent(operations, "", "    ")
	reply(log, w, nil, b, err)
}
