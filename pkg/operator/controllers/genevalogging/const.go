package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

const (
	kubeNamespace         = "openshift-azure-logging"
	kubeServiceAccount    = "system:serviceaccount:" + kubeNamespace + ":geneva"
	otelConfigMapName     = "otel-config"
	otelMasterConfigKey   = "master-config.yaml"
	otelWorkerConfigKey   = "worker-config.yaml"
	otelGatewayCACMName   = "gateway-ca-otel-export"
	legacyGatewayCACMName = "gateway-ca"
	otelGatewayCAKey      = "ca-bundle.crt"
)

//go:embed staticfiles/otel-config.yaml
var otelConfigHighLogLevel string

//go:embed staticfiles/otel-config-reduced-noise.yaml
var otelConfigReducedLogs string

//go:embed staticfiles/otel-config-high-signal.yaml
var otelConfigMinimalLogs string

