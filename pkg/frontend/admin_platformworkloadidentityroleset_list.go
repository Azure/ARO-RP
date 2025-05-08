package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (f *frontend) getAdminPlatformWorkloadIdentityRoleSets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	converter := f.apis[admin.APIVersion].PlatformWorkloadIdentityRoleSetConverter

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

	var roleSets []*api.PlatformWorkloadIdentityRoleSet
	if docs != nil {
		for _, doc := range docs.PlatformWorkloadIdentityRoleSetDocuments {
			roleSets = append(roleSets, doc.PlatformWorkloadIdentityRoleSet)
		}
	}

	sort.Slice(roleSets, func(i, j int) bool {
		return version.CreateSemverFromMinorVersionString(roleSets[i].Properties.OpenShiftVersion).LessThan(*version.CreateSemverFromMinorVersionString(roleSets[j].Properties.OpenShiftVersion))
	})

	b, err := json.MarshalIndent(converter.ToExternalList(roleSets), "", "    ")
	adminReply(log, w, nil, b, err)
}
