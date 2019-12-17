package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/frontend/middleware"
)

func (f *frontend) postOpenShiftClusterCredentials(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) > 0 && !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getOpenShiftCluster(r, api.APIs[vars["api-version"]]["OpenShiftClusterCredentials"].(api.OpenShiftClusterToExternal))

	reply(log, w, b, err)
}
