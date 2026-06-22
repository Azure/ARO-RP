package mimo

import "github.com/Azure/ARO-RP/pkg/api"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	TLS_CERT_ROTATION_ID      api.MIMOTaskID = "9b741734-6505-447f-8510-85eb0ae561a2"
	OPERATOR_VERSION_RESET_ID api.MIMOTaskID = "75ab83d0-9a56-4ae3-843d-0db43679b082"
	OPERATOR_FLAGS_UPDATE_ID  api.MIMOTaskID = "b41749fc-af26-4ab7-b5a1-e03f3ee4cba6"
	ACR_TOKEN_CHECKER_ID      api.MIMOTaskID = "082978ce-3700-4972-835f-53d48658d291"
	MSI_CERT_RENEWAL_ID       api.MIMOTaskID = "7c3f8e2d-9a4b-4f1e-8c5d-2b6a9e7f3d1c"
	MIGRATE_LB_ZONES_ID       api.MIMOTaskID = "c28a07c1-462f-42d2-8031-4b0222256596"
	FIX_SSH_ID                api.MIMOTaskID = "888fc221-b059-49db-bd02-22ad86cccd6b"

	// Operator Flag setting tasks
	OPERATOR_FLAG_SET_GENEVA_OTEL                      api.MIMOTaskID = "eb0360af-4748-42a3-9788-dfffae58dff6"
	OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_MAX_LOGS     api.MIMOTaskID = "fbfa4ce6-117a-4c2e-8317-0834e51d6f6f"
	OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_REDUCED_LOGS api.MIMOTaskID = "62e61118-78ff-4d11-abaa-e171ae1edd39"
	OPERATOR_FLAG_SET_GENEVA_OTEL_PROFILE_MINIMAL_LOGS api.MIMOTaskID = "59444b6f-c5e1-4d12-84b2-81d8ed5af9c9"
)
