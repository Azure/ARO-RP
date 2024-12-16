package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getPlatformWorkloadIdentityRoleSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
	requestedMinorVersion := chi.URLParam(r, "openShiftMinorVersion")
	if f.apis[apiVersion].PlatformWorkloadIdentityRoleSetConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The endpoint could not be found in the namespace '%s' for api version '%s'.", resourceProviderNamespace, apiVersion)
		return
	}

	f.platformWorkloadIdentityRoleSetsMu.RLock()
	platformWorkloadIdentityRoleSet, ok := f.availablePlatformWorkloadIdentityRoleSets[requestedMinorVersion]
	f.platformWorkloadIdentityRoleSetsMu.RUnlock()
	if !ok {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceNotFound, "", "The Resource platformWorkloadIdentityRoleSet with version '%s' was not found in the namespace '%s' for api version '%s'.", requestedMinorVersion, resourceProviderNamespace, apiVersion)
		return
	}

	converter := f.apis[apiVersion].PlatformWorkloadIdentityRoleSetConverter

	b, err := json.MarshalIndent(converter.ToExternal(platformWorkloadIdentityRoleSet), "", "    ")
	frontendOperationResultLog(log, r.Method, err)
	reply(log, w, nil, b, err)
}
