// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package genevalogging

import (
	"bytes"
	"fmt"
	"text/template"
)

var otelConfigParsedTemplate = template.Must(template.New("otel-config").Parse(otelConfigTemplate))

func renderOTelConfig(profile otelProfile) (string, error) {
	var rendered bytes.Buffer
	err := otelConfigParsedTemplate.Execute(&rendered, struct {
		Profile           otelProfile
		GatewayExporterID string
	}{
		Profile:           profile,
		GatewayExporterID: "otlp_grpc/gateway",
	})
	if err != nil {
		return "", fmt.Errorf("failed to render otel config template for profile %q: %w", profile, err)
	}

	return rendered.String(), nil
}
