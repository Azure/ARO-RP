package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (f *frontend) listInstallVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	if f.apis[apiVersion].OpenShiftVersionConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The endpoint could not be found in the namespace '%s' for api version '%s'.", vars["resourceProviderNamespace"], apiVersion)
		return
	}

	versions := f.getEnabledInstallVersions(ctx)
	converter := f.apis[apiVersion].OpenShiftVersionConverter

	b, err := json.MarshalIndent(converter.ToExternalList(versions), "", "    ")
	reply(log, w, nil, b, err)
}

func (f *frontend) getEnabledInstallVersions(ctx context.Context) []*api.OpenShiftVersion {
	versions := make([]*api.OpenShiftVersion, 0)

	f.mu.RLock()
	for _, v := range f.enabledOcpVersions {
		versions = append(versions, v)
	}
	f.mu.RUnlock()

	if len(versions) == 0 {
		versions = append(versions, &api.OpenShiftVersion{
			Properties: api.OpenShiftVersionProperties{
				Version: version.InstallStream.Version.String(),
			},
		})
	}

	return versions
}
