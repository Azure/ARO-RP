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
	legacyGatewayCACMName = "gateway-ca"
)

//go:embed staticfiles/otel-config.yaml.tmpl
var otelConfigTemplate string
