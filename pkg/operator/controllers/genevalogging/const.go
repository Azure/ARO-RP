package genevalogging

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import _ "embed"

const (
	kubeNamespace          = "openshift-azure-logging"
	kubeServiceAccount     = "system:serviceaccount:" + kubeNamespace + ":geneva"
	certificatesSecretName = "certificates"

	GenevaCertName = "gcscert.pem"
	GenevaKeyName  = "gcskey.pem"
)

//go:embed staticfiles/fluent.conf
var fluentConf string

//go:embed staticfiles/parsers.conf
var parsersConf string
