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

const (
	DEFAULT_POLL_TIME        = time.Second * 10
	DEFAULT_TIMEOUT_DURATION = time.Minute * 20
)

var DEFAULT_MAINTENANCE_TASKS = map[api.MIMOTaskID]MaintenanceTask{
	mimo.TLS_CERT_ROTATION_ID:      TLSCertRotation,
	mimo.ACR_TOKEN_CHECKER_ID:      ACRTokenChecker,
	mimo.OPERATOR_FLAGS_UPDATE_ID:  UpdateOperatorFlags,
	mimo.MDSD_CERT_ROTATION_ID:     MDSDCertRotation,
	mimo.MSI_CERT_RENEWAL_ID:       MSICertificateRenewal,
	mimo.CONCERNING_CSV_CHECKER_ID: ConcerningCSVChecker,
}

func run(t utilmimo.TaskContext, s []steps.Step) error {
	_, err := steps.Run(t, t.Log(), DEFAULT_POLL_TIME, s, t.Now)
	return err
}
