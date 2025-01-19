package trace

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"time"

	sdklogging "github.com/openshift-online/ocm-sdk-go/logging"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
)

func InstallOpenTelemetryTracer(ctx context.Context, logger sdklogging.Logger, resourceAttrs ...attribute.KeyValue) (
	func(context.Context) error,
	error,
) {
	// NOTE: Auto span exporter sends traces to https://localhost:4318/v1/traces by default.
	// We overwrite the default value with "none".
	// See:
	// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/f6667f6f9eab2370f46d0903cf323cda3b7ca2bd/exporters/autoexport/spans.go#L32-L60
	if v, _ := os.LookupEnv("OTEL_TRACES_EXPORTER"); v == "" {
		os.Setenv("OTEL_TRACES_EXPORTER", "none")
	}
	logger.Info(ctx, "initialising OpenTelemetry tracer")

	exp, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTEL exporter: %w", err)
	}

	opts := []resource.Option{resource.WithHost()}
	if len(resourceAttrs) > 0 {
		opts = append(opts, resource.WithAttributes(resourceAttrs...))
	}
	resources, err := resource.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise trace resources: %w", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resources),
	)
	otel.SetTracerProvider(tp)

	shutdown := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}

	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)

	otel.SetErrorHandler(otelErrorHandlerFunc(func(err error) {
		logger.Error(ctx, "OpenTelemetry.ErrorHandler: %v", err)
	}))

	return shutdown, nil
}

type otelErrorHandlerFunc func(error)

// Handle implements otel.ErrorHandler
func (f otelErrorHandlerFunc) Handle(err error) {
	f(err)
}
