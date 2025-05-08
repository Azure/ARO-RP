package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) listInstallVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
	if f.apis[apiVersion].OpenShiftVersionConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", fmt.Sprintf("The endpoint could not be found in the namespace '%s' for api version '%s'.", resourceProviderNamespace, apiVersion))
		return
	}

	versions := f.getEnabledInstallVersions(ctx)
	converter := f.apis[apiVersion].OpenShiftVersionConverter

	b, err := json.MarshalIndent(converter.ToExternalList(versions), "", "    ")
	reply(log, w, nil, b, err)
}

func (f *frontend) getEnabledInstallVersions(ctx context.Context) []*api.OpenShiftVersion {
	versions := make([]*api.OpenShiftVersion, 0)

	f.ocpVersionsMu.RLock()
	for _, v := range f.enabledOcpVersions {
		versions = append(versions, v)
	}
	f.ocpVersionsMu.RUnlock()

	return versions
}
