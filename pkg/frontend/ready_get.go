package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

// checkReady checks the ready status of the frontend to make it consistent
// across the /healthz/ready endpoint and emitted metrics.   We wait for 2
// minutes before indicating health.  This ensures that there will be a gap in
// our health metric if we crash or restart.
func (f *frontend) checkReady() bool {
	if !f.env.FeatureIsSet(env.FeatureDisableReadinessDelay) &&
		time.Since(f.startTime) < 2*time.Minute {
		return false
	}

	_, okOcpVersions := f.lastOcpVersionsChangefeed.Load().(time.Time)
	_, okPlatformWorkloadIdentityRoleSets := f.lastPlatformWorkloadIdentityRoleSetsChangefeed.Load().(time.Time)

	var miseAuthReady, armAuthReady, authReady bool
	if f.authMiddleware.EnableMISE {
		miseAuthReady = f.env.MISEAuthorizer().IsReady()
	}
	// skip ARM Authorizer is MISE is Enforcing
	if !f.authMiddleware.EnforceMISE {
		armAuthReady = f.env.ArmClientAuthorizer().IsReady()
	}
	authReady = miseAuthReady || armAuthReady
	return okOcpVersions && okPlatformWorkloadIdentityRoleSets &&
		f.ready.Load().(bool) &&
		authReady &&
		f.env.AdminClientAuthorizer().IsReady()
}

func (f *frontend) getReady(w http.ResponseWriter, r *http.Request) {
	if f.checkReady() {
		api.WriteCloudError(w, &api.CloudError{StatusCode: http.StatusOK})
	} else {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
	}
}
