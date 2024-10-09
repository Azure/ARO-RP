package tasks

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/mimo"
	utilmimo "github.com/Azure/ARO-RP/pkg/util/mimo"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

const DEFAULT_POLL_TIME = time.Second * 10
const DEFAULT_TIMEOUT_DURATION = time.Minute * 20

var DEFAULT_MAINTENANCE_SETS = map[string]MaintenanceTask{
	mimo.TLS_CERT_ROTATION_ID:     TLSCertRotation,
	mimo.ACR_TOKEN_CHECKER_ID:     ACRTokenChecker,
	mimo.OPERATOR_FLAGS_UPDATE_ID: UpdateOperatorFlags,
}

func run(t utilmimo.TaskContext, s []steps.Step) (api.MaintenanceManifestState, string) {
	_, err := steps.Run(t, t.Log(), DEFAULT_POLL_TIME, s, t.Now)

	if err != nil {
		if utilmimo.IsRetryableError(err) {
			return api.MaintenanceManifestStatePending, err.Error()
		}
		return api.MaintenanceManifestStateFailed, err.Error()
	}
	return api.MaintenanceManifestStateCompleted, t.GetResultMessage()
}
