// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

package genevalogging

import (
	"bytes"
	"fmt"
	"text/template"
)

var otelConfigParsedTemplate = template.Must(template.New("otel-config").Parse(otelConfigTemplate))

type otelLogSource struct {
	Name      string
	Receiver  string
	EventName string
}

func renderOTelConfig(profile otelProfile, emitSourceFields bool) (string, error) {
	var rendered bytes.Buffer
	err := otelConfigParsedTemplate.Execute(&rendered, struct {
		Profile           otelProfile
		EmitSourceFields  bool
		GatewayExporterID string
		Sources           []otelLogSource
	}{
		Profile:           profile,
		EmitSourceFields:  emitSourceFields,
		GatewayExporterID: "otlp_grpc/gateway",
		Sources: []otelLogSource{
			{
				Name:      "journald",
				Receiver:  "journald",
				EventName: "journald",
			},
			{
				Name:      "containers",
				Receiver:  "file_log/containers",
				EventName: "containers",
			},
			{
				Name:      "audit",
				Receiver:  "file_log/audit",
				EventName: "audit",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to render otel config template for profile %q: %w", profile, err)
	}

	return rendered.String(), nil
}
