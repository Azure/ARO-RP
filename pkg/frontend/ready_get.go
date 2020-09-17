package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

// checkReady checks the ready status of the frontend to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (f *frontend) checkReady() bool {
	if f.env.DeploymentMode() != deployment.Development &&
		time.Now().Sub(f.startTime) < 2*time.Minute {
		return false
	}

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
