package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
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

	if f.apis[vars["api-version"]].OpenShiftVersionConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The endpoint could not be found in the namespace '%s' for api version '%s'.", vars["resourceProviderNamespace"], vars["api-version"])
		return
	}

	versions, err := f.getEnabledInstallVersions(ctx)
	if err != nil {
		log.Error(err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Unable to list the available OpenShift versions in this region.")
		return
	}

	converter := f.apis[vars["api-version"]].OpenShiftVersionConverter

	b, err := json.MarshalIndent(converter.ToExternalList(([]*api.OpenShiftVersion)(versions)), "", "    ")
	reply(log, w, nil, b, err)
}

func (f *frontend) getEnabledInstallVersions(ctx context.Context) ([]*api.OpenShiftVersion, error) {
	docs, err := f.dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to list the entries in the OpenShift versions database: %s", err.Error())
	}

	versions := make([]*api.OpenShiftVersion, 0)
	for _, doc := range docs.OpenShiftVersionDocuments {
		if doc.OpenShiftVersion.Properties.Enabled {
			versions = append(versions, doc.OpenShiftVersion)
		}
	}

	// add the default from version.InstallStream, when we have no active versions
	if len(versions) == 0 {
		versions = append(versions, &api.OpenShiftVersion{
			Properties: api.OpenShiftVersionProperties{
				Version: version.InstallStream.Version.String(),
			},
		})
	}

	return versions, nil
}
