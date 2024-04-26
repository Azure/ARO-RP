package api

import (
	"time"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// CorrelationData represents any data, used for metrics or tracing.
// More on these values:
// https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/common-api-details.md
type CorrelationData struct {
	// CorrelationID contains value of x-ms-correlation-request-id
	CorrelationID string `json:"correlationId,omitempty"`

	// ClientRequestID contains value of x-ms-client-request-id
	ClientRequestID string `json:"clientRequestId,omitempty"`

	// RequestID contains value of x-ms-request-id
	RequestID string `json:"requestID,omitempty"`

	// ClientPrincipalName contains value of x-ms-client-principal-name
	ClientPrincipalName string `json:"clientPrincipalName,omitempty"`

	// RequestTime is the time that the request was received
	RequestTime time.Time `json:"requestTime,omitempty"`
}
