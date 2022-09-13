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

	converter := f.apis[admin.APIVersion].OpenShiftVersionConverter
	staticValidator := f.apis[admin.APIVersion].OpenShiftVersionStaticValidator

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 || !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	var version *admin.OpenShiftVersion

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

	var versionDoc *api.OpenShiftVersionDocument

	if docs != nil {
		for _, doc := range docs.OpenShiftVersionDocuments {
			if doc.OpenShiftVersion.Version == version.Version {
				versionDoc = doc
				break
			}
		}
	}

	isCreate := false
	if versionDoc == nil {
		isCreate = true
		err = staticValidator.Static(version, nil)
		versionDoc = &api.OpenShiftVersionDocument{
			ID:               f.dbOpenShiftVersions.NewUUID(),
			OpenShiftVersion: &api.OpenShiftVersion{},
		}
	} else {
		err = staticValidator.Static(version, versionDoc.OpenShiftVersion)
	}
	if err != nil {
		adminReply(log, w, nil, []byte{}, err)
		return
	}

	converter.ToInternal(version, versionDoc.OpenShiftVersion)

	if isCreate {
		versionDoc, err = f.dbOpenShiftVersions.Create(ctx, versionDoc)
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	} else {
		versionDoc, err = f.dbOpenShiftVersions.Update(ctx, versionDoc)
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	}

	b, err := json.MarshalIndent(converter.ToExternal(versionDoc.OpenShiftVersion), "", "    ")
	if err == nil {
		if isCreate {
			err = statusCodeError(http.StatusCreated)
		}
	}
	adminReply(log, w, nil, b, err)
}
