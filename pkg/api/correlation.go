package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type contextKey int

const (
	contextKeyCorrelationData contextKey = iota
)

// CorrelationData represents data used for metrics or tracing between ARO-RP and ARM.
// https://eng.ms/docs/products/arm/rpaas/contract/requestheaders
type CorrelationData struct {
	// CorrelationID contains value of x-ms-correlation-request-id
	CorrelationID string `json:"correlationId,omitempty"`

	// ClientRequestID contains value of x-ms-client-request-id
	ClientRequestID string `json:"clientRequestId,omitempty"`

	// OperationID contains the unique ID generated for each operation that the ARO-RP performs
	OperationID string `json:"operationID,omitempty"`

	// RequestID contains value of x-ms-request-id
	RequestID string `json:"requestID,omitempty"`

	// ClientPrincipalName contains value of x-ms-client-principal-name
	ClientPrincipalName string `json:"clientPrincipalName,omitempty"`

	// RequestTime is the time that the request was received
	RequestTime time.Time `json:"requestTime,omitempty"`
}

func CtxWithCorrelationData(ctx context.Context, correlationData *CorrelationData) context.Context {
	return context.WithValue(ctx, contextKeyCorrelationData, correlationData)
}

func GetCorrelationDataFromCtx(ctx context.Context) *CorrelationData {
	correlationData, ok := ctx.Value(contextKeyCorrelationData).(*CorrelationData)
	if !ok {
		return nil
	}
	return correlationData
}

func CreateCorrelationDataFromReq(req *http.Request) *CorrelationData {
	if req == nil {
		return nil
	}

	return &CorrelationData{
		ClientRequestID: req.Header.Get("X-Ms-Client-Request-Id"),
		CorrelationID:   req.Header.Get("X-Ms-Correlation-Request-Id"),
		RequestID:       uuid.DefaultGenerator.Generate(),
		OperationID:     uuid.DefaultGenerator.Generate(),
	}
}
