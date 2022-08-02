package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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

	if f.apis[vars["api-version"]].InstallVersionsConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The endpoint could not be found in the namespace '%s' for api version '%s'.", vars["resourceProviderNamespace"], vars["api-version"])
		return
	}

	versions := f.getInstallVersions()
	converter := f.apis[vars["api-version"]].InstallVersionsConverter()

	b, err := json.Marshal(converter.ToExternal((*api.InstallVersions)(&versions)))
	reply(log, w, nil, b, err)
}

// Creating this separate method so that it can be reused while doing some other stuff like validating the install version against available version.
func (f *frontend) getInstallVersions() []string {
	// TODO: Currently, the versions are hard coded and being pulled from version.UpgradeStreams, but in future they will be pulled from cosmosdb.
	installStream := version.InstallStream
	versions := make([]string, 0)
	versions = append(versions, installStream.Version.String())

	return versions
}
