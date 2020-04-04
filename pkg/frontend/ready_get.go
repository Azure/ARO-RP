package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
)

//checkReady checks the ready status of the frontend to make it consistent across the /healthz/ready endpoint and emited metrics
func (f *frontend) checkReady() bool {
	return f.ready.Load().(bool) &&
		f.env.ArmClientAuthorizer().IsReady() &&
		f.env.AdminClientAuthorizer().IsReady()
}

func (f *frontend) getReady(w http.ResponseWriter, r *http.Request) {
	if f.checkReady() {
		api.WriteCloudError(w, &api.CloudError{StatusCode: http.StatusOK})
	} else {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
	}
}
