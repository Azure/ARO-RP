package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getInstallVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
	requestedVersion := chi.URLParam(r, "openshiftVersion")
	if f.apis[apiVersion].OpenShiftVersionConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", fmt.Sprintf("The endpoint could not be found in the namespace '%s' for api version '%s'.", resourceProviderNamespace, apiVersion))
		return
	}

	f.ocpVersionsMu.RLock()
	version, ok := f.enabledOcpVersions[requestedVersion]
	f.ocpVersionsMu.RUnlock()
	if !ok {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource openShiftVersion with version '%s' was not found in the namespace '%s' for api version '%s'.", requestedVersion, resourceProviderNamespace, apiVersion))
		return
	}

	converter := f.apis[apiVersion].OpenShiftVersionConverter

	b, err := json.MarshalIndent(converter.ToExternal(version), "", "    ")
	frontendOperationResultLog(log, r.Method, err)
	reply(log, w, nil, b, err)
}
