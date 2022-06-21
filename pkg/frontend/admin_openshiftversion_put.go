package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putAdminOpenShiftVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	converter := f.apis[admin.APIVersion].OpenShiftVersionConverter()

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 || !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	var version admin.OpenShiftVersion

	err := json.Unmarshal(body, &version)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content could not be deserialized: "+err.Error())
		return
	}

	docs, err := f.dbOpenShiftVersions.ListAll(ctx)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		return
	}

	var foundDoc *api.OpenShiftVersionDocument
	var returnDoc *api.OpenShiftVersionDocument

	if docs != nil {
		for _, doc := range docs.OpenShiftVersionDocuments {
			if doc.OpenShiftVersion.Version == version.Version {
				foundDoc = doc
				break
			}
		}
	}

	isCreate := false
	if foundDoc != nil {
		returnDoc, err = f.dbOpenShiftVersions.Patch(ctx, foundDoc.ID, func(osvd *api.OpenShiftVersionDocument) error {
			osvd.OpenShiftVersion.OpenShiftPullspec = version.OpenShiftPullspec
			osvd.OpenShiftVersion.InstallerPullspec = version.InstallerPullspec
			osvd.OpenShiftVersion.Enabled = version.Enabled
			return nil
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	} else {
		isCreate = true
		returnDoc, err = f.dbOpenShiftVersions.Create(ctx, &api.OpenShiftVersionDocument{
			ID: f.dbOpenShiftVersions.NewUUID(),
			OpenShiftVersion: &api.OpenShiftVersion{
				Version:           version.Version,
				OpenShiftPullspec: version.OpenShiftPullspec,
				InstallerPullspec: version.InstallerPullspec,
				Enabled:           version.Enabled,
			},
		})
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	}

	b, err := json.MarshalIndent(converter.ToExternal(returnDoc.OpenShiftVersion), "", "    ")
	if err == nil {
		if isCreate {
			err = statusCodeError(http.StatusCreated)
		}
	}
	adminReply(log, w, nil, b, err)
}
