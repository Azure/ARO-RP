package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/portal/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type PortalInfo struct {
	Location  string `json:"location"`
	Elevated  bool   `json:"elevated"`
	Username  string `json:"username"`
	RPVersion string `json:"rpversion"`
}

func (p *portal) info(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	elevated := len(middleware.GroupsIntersect(p.elevatedGroupIDs, ctx.Value(middleware.ContextKeyGroups).([]string))) > 0

	resp := PortalInfo{
		Location:  p.env.Location(),
		Elevated:  elevated,
		Username:  ctx.Value(middleware.ContextKeyUsername).(string),
		RPVersion: version.GitCommit,
	}

	b, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
