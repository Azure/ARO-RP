package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

const (
	kubeNamespace          = "openshift-azure-logging"
	kubeServiceAccount     = "system:serviceaccount:" + kubeNamespace + ":geneva"
	certificatesSecretName = "certificates"
	otelConfigMapName      = "otel-config"
	otelMasterConfigKey    = "master-config.yaml"
	otelWorkerConfigKey    = "worker-config.yaml"
	otelGatewayCACMName    = "gateway-ca-otel-export"
	legacyGatewayCACMName  = "gateway-ca"
	otelGatewayCAKey       = "ca-bundle.crt"

	GenevaCertName = "gcscert.pem"
	GenevaKeyName  = "gcskey.pem"
)

//go:embed staticfiles/fluent.conf
var fluentConf string

//go:embed staticfiles/parsers.conf
var parsersConf string

//go:embed staticfiles/otel-config.yaml
var otelConfigHighLogLevel string

//go:embed staticfiles/otel-config-reduced-noise.yaml
var otelConfigReducedLogs string

//go:embed staticfiles/otel-config-high-signal.yaml
var otelConfigMinimalLogs string

// Backward-compatible aliases for existing tests and references.
var (
	otelConfigFull         = otelConfigHighLogLevel
	otelConfigReducedNoise = otelConfigReducedLogs
	otelConfigHighSignal   = otelConfigMinimalLogs
)
