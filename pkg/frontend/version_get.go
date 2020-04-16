package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type rpVersion struct {
	Provider  string `json:"provider"`
	OpenShift string `json:"openshift"`
}

func (f *frontend) getVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	b, err := json.Marshal(rpVersion{
		Provider:  version.ProviderVersion,
		OpenShift: version.OpenShiftVersion,
	})

	reply(log, w, nil, b, err)
}
