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

func (f *frontend) listPlatformWorkloadIdentityRoleSets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	apiVersion := r.URL.Query().Get(api.APIVersionKey)
	resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
	if f.apis[apiVersion].PlatformWorkloadIdentityRoleSetConverter == nil {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The endpoint could not be found in the namespace '%s' for api version '%s'.", resourceProviderNamespace, apiVersion)
		return
	}

	roleSets := f.getAvailablePlatformWorkloadIdentityRoleSets()
	converter := f.apis[apiVersion].PlatformWorkloadIdentityRoleSetConverter

	b, err := json.MarshalIndent(converter.ToExternalList(roleSets), "", "    ")
	reply(log, w, nil, b, err)
}

func (f *frontend) getAvailablePlatformWorkloadIdentityRoleSets() []*api.PlatformWorkloadIdentityRoleSet {
	roleSets := make([]*api.PlatformWorkloadIdentityRoleSet, 0)

	f.platformWorkloadIdentityRoleSetsMu.RLock()
	for _, pwirs := range f.availablePlatformWorkloadIdentityRoleSets {
		roleSets = append(roleSets, pwirs)
	}
	f.platformWorkloadIdentityRoleSetsMu.RUnlock()

	return roleSets
}
