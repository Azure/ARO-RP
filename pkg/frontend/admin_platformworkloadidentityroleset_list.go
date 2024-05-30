package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"

	"github.com/coreos/go-semver/semver"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminPlatformWorkloadIdentityRoleSets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	converter := f.apis[admin.APIVersion].PlatformWorkloadIdentityRoleSetConverter

	docs, err := f.dbPlatformWorkloadIdentityRoleSets.ListAll(ctx)
	if err != nil {
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
		appendPatch := func(in string) string {
			return in + ".0"
		}
		return semver.New(appendPatch(roleSets[i].Properties.OpenShiftVersion)).LessThan(*semver.New(appendPatch(roleSets[j].Properties.OpenShiftVersion)))
	})

	b, err := json.MarshalIndent(converter.ToExternalList(roleSets), "", "    ")
	adminReply(log, w, nil, b, err)
}
