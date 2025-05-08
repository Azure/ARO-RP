package azure

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestTracer(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	m.EXPECT().EmitGauge("client.azure.duration", gomock.Any(), map[string]string{
		"client": "test",
		"code":   "401",
	})
	m.EXPECT().EmitGauge("client.azure.count", int64(1), map[string]string{
		"client": "test",
		"code":   "401",
	})
	m.EXPECT().EmitGauge("client.azure.errors", int64(1), map[string]string{
		"client": "test",
		"code":   "401",
	})

	tr := New(m)
	ctx := tr.StartSpan(context.Background(), "test")
	tr.EndSpan(ctx, http.StatusUnauthorized, errors.New("authorization failed"))
}
