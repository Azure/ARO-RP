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

func (f *frontend) putAdminPlatformWorkloadIdentityRoleSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	converter := f.apis[admin.APIVersion].PlatformWorkloadIdentityRoleSetConverter
	staticValidator := f.apis[admin.APIVersion].PlatformWorkloadIdentityRoleSetStaticValidator

	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	if len(body) == 0 || !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized.")
		return
	}

	var ext *admin.PlatformWorkloadIdentityRoleSet
	err := json.Unmarshal(body, &ext)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content could not be deserialized: "+err.Error())
		return
	}

	dbPlatformWorkloadIdentityRoleSets, err := f.dbGroup.PlatformWorkloadIdentityRoleSets()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
		return
	}

	docs, err := dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
		log.Error(err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		return
	}

	var roleSetDoc *api.PlatformWorkloadIdentityRoleSetDocument
	if docs != nil {
		for _, doc := range docs.PlatformWorkloadIdentityRoleSetDocuments {
			if doc.PlatformWorkloadIdentityRoleSet.Properties.OpenShiftVersion == ext.Properties.OpenShiftVersion {
				roleSetDoc = doc
				break
			}
		}
	}

	isCreate := roleSetDoc == nil
	if isCreate {
		err = staticValidator.Static(ext, nil)
		roleSetDoc = &api.PlatformWorkloadIdentityRoleSetDocument{
			ID:                              dbPlatformWorkloadIdentityRoleSets.NewUUID(),
			PlatformWorkloadIdentityRoleSet: &api.PlatformWorkloadIdentityRoleSet{},
		}
	} else {
		err = staticValidator.Static(ext, roleSetDoc.PlatformWorkloadIdentityRoleSet)
	}
	if err != nil {
		adminReply(log, w, nil, []byte{}, err)
		return
	}

	converter.ToInternal(ext, roleSetDoc.PlatformWorkloadIdentityRoleSet)

	if isCreate {
		roleSetDoc, err = dbPlatformWorkloadIdentityRoleSets.Create(ctx, roleSetDoc)
		if err != nil {
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	} else {
		roleSetDoc, err = dbPlatformWorkloadIdentityRoleSets.Update(ctx, roleSetDoc)
		if err != nil {
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	}

	b, err := json.MarshalIndent(converter.ToExternal(roleSetDoc.PlatformWorkloadIdentityRoleSet), "", "    ")
	if err == nil {
		if isCreate {
			err = statusCodeError(http.StatusCreated)
		}
	}
	adminReply(log, w, nil, b, err)
}
