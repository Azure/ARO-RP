package k8s

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestTracer(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := mock_metrics.NewMockEmitter(controller)

	m.EXPECT().EmitGauge("client.k8s.count", int64(1), map[string]string{
		"verb": "GET",
		"code": "<error>",
	})
	m.EXPECT().EmitGauge("client.k8s.errors", int64(1), map[string]string{
		"verb": "GET",
		"code": "<error>",
	})

	tr := &tracer{
		m: m,
	}
	tr.Increment(context.Background(), "<error>", "GET", "host")
}
