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

func renderOTelConfig(profile otelProfile) (string, error) {
	return renderOTelConfigWithAudit(profile, true)
}

func renderOTelConfigWithoutAudit(profile otelProfile) (string, error) {
	return renderOTelConfigWithAudit(profile, false)
}

func renderOTelConfigWithAudit(profile otelProfile, isControlPlane bool) (string, error) {
	sources := []otelLogSource{
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
	}
	if isControlPlane {
		sources = append(sources, otelLogSource{
			Name:      "audit",
			Receiver:  "file_log/audit",
			EventName: "audit",
		})
	}

	var rendered bytes.Buffer
	err := otelConfigParsedTemplate.Execute(&rendered, struct {
		Profile           otelProfile
		GatewayExporterID string
		IsControlPlane    bool
		Sources           []otelLogSource
	}{
		Profile:           profile,
		GatewayExporterID: "otlp_grpc/gateway",
		IsControlPlane:    isControlPlane,
		Sources:           sources,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render otel config template for profile %q: %w", profile, err)
	}

	return rendered.String(), nil
}
